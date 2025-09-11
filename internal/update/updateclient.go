// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/foundriesio/composeapp/pkg/compose"
	v1 "github.com/foundriesio/composeapp/pkg/compose/v1"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fiotuf/tuf"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/internal/targets"
	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	_ "modernc.org/sqlite"
)

// Atributes of the UpdateContext instance are gradually set during the update process
type (
	UpdateContext struct {
		opts       *UpdateOptions
		DbFilePath string

		Target             *metadata.TargetFiles
		CurrentTarget      *metadata.TargetFiles
		Reason             string
		RequiredApps       []string
		AppsToUninstall    []string
		InstalledApps      []string
		InstalledAppsNames []string
		ConfiguredAppNames []string
		TargetIsRunning    bool

		Context       context.Context
		ComposeConfig *compose.Config
		Runner        update.Runner

		PendingRunner     update.Runner
		PendingTargetName string
		PendingApps       []string

		Resuming bool
	}
)

func InitializeDatabase(dbFilePath string) error {
	err := targets.CreateTargetsTable(dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to create targets table %w", err)
	}

	err = events.CreateEventsTable(dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to create events table %w", err)
	}

	return nil
}

func getTargetsTuf(config *sotatoml.AppConfig, localRepoPath string, client *http.Client, refreshTargets bool, currentTargetName string, appsNames []string) (map[string]*metadata.TargetFiles, error) {
	// TODO: Set currentTargetName and appsNames in Fiotuf instance, for it to update the x-ats-* headers accordingly
	fiotuf, err := tuf.NewFioTuf(config, client)
	if err != nil {
		log.Err(err).Msg("Error creating fiotuf instance")
		return nil, err
	}

	if refreshTargets {
		err = fiotuf.RefreshTuf(localRepoPath)
		if err != nil {
			log.Err(err).Msg("Error refreshing TUF")
			return nil, err
		}
	}

	tufTargets := fiotuf.GetTargets()
	return tufTargets, nil
}

func fetchTargetsJson(config *sotatoml.AppConfig, client *http.Client, currentTargetName string, appsNames []string) ([]byte, error) {
	urlPath := config.GetDefault("tls.server", "https://ota-lite.foundries.io:8443") + "/repo/targets.json"
	headers := make(map[string]string)
	headers["x-ats-tags"] = config.Get("pacman.tags")
	headers["x-ats-target"] = currentTargetName
	headers["x-ats-dockerapps"] = strings.Join(appsNames, ",")
	res, err := transport.HttpGet(client, urlPath, headers)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", res.StatusCode, urlPath)
	}
	return res.Body, nil
}

func getTargetsUnsafe(config *sotatoml.AppConfig, localRepoPath string, client *http.Client, refreshTargets bool, currentTargetName string, appsNames []string) (map[string]*metadata.TargetFiles, error) {
	var targetsBytes []byte
	var err error

	// Store unverified targets.json outside "tuf" directory
	unsafeTargetsPath := path.Join(config.GetDefault("storage.path", "/var/sota"), "targets.json")
	if refreshTargets {
		if localRepoPath == "" {
			targetsBytes, err = fetchTargetsJson(config, client, currentTargetName, appsNames)
			if err != nil {
				return nil, fmt.Errorf("error fetching targets.json: %w", err)
			}
		} else {
			filePath := path.Join(localRepoPath, "repo", "targets.json")
			targetsBytes, err = os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("error reading targets.json from %s: %v", filePath, err)
			}
		}
		// Write contents of targetsBytes to local file
		err = os.WriteFile(unsafeTargetsPath, targetsBytes, 0644)
		if err != nil {
			return nil, fmt.Errorf("error writing targets.json to %s: %w", unsafeTargetsPath, err)
		}
	} else {
		targetsBytes, err = os.ReadFile(unsafeTargetsPath)
		if err != nil {
			return nil, fmt.Errorf("error reading targets.json from %s: %w", unsafeTargetsPath, err)
		}
	}

	meta, err := metadata.Targets().FromBytes(targetsBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing targets.json: %w", err)
	}
	targets := meta.Signed.Targets

	return targets, nil

}

func getTargets(config *sotatoml.AppConfig, localRepoPath string, client *http.Client, currentTargetName string, appsNames []string, enableTuf bool, refreshTargets bool) (map[string]*metadata.TargetFiles, error) {
	if enableTuf {
		return getTargetsTuf(config, localRepoPath, client, refreshTargets, currentTargetName, appsNames)
	} else {
		return getTargetsUnsafe(config, localRepoPath, client, refreshTargets, currentTargetName, appsNames)
	}
}

func checkUpdateState(updateContext *UpdateContext, targetId string) error {
	log.Debug().Msgf("Checking update state. targetId: %s", targetId)
	// standalone check command has no state requirements
	if updateContext.opts.DoCheck && !updateContext.opts.DoFetch {
		log.Debug().Msg("Standalone check command, no state requirements")
		return nil
	}

	var updateState update.State
	if updateContext.PendingRunner != nil {
		updateState = updateContext.PendingRunner.Status().State
	}

	// standalone install and start commands require a pending update operation at the right state
	if (updateContext.opts.DoInstall || updateContext.opts.DoStart) && !updateContext.opts.DoFetch {
		log.Debug().Msg("Standalone install or start command, checking requirements")
		if updateContext.PendingRunner == nil {
			return fmt.Errorf("no pending target to perform operation on")
		}
		if updateContext.opts.DoInstall {
			// Check valid states for standalone install command
			if updateState != update.StateFetched && updateState != update.StateInstalling {
				return fmt.Errorf("cannot install, current update is in '%s' state", updateState.String())
			}
		} else {
			// Check valid states for standalone start command
			if updateState != update.StateInstalled && updateState != update.StateStarting && updateState != update.StateStarted && updateState != update.StateCompleting {
				return fmt.Errorf("cannot start, current update is in '%s' state", updateState.String())
			}
		}
		return nil
	}

	// update and standalone fetch commands requires that either:
	// - there is no pending update; or
	// - no targetId was specified by the user, so we proceed with whatever update was going on; or
	// - the pending update matches the targetId selected by the user
	if updateContext.opts.DoFetch && updateContext.PendingRunner != nil {
		log.Debug().Msg("Update or standalone fetch command, checking requirements")
		if targetId != "" {
			if _, err := strconv.Atoi(targetId); err == nil {
				// targetId is a version, check if PendingTargetName ends with -<version>
				log.Debug().Msg("targetId is a version, checking if PendingTargetName ends with -<version>")
				if !strings.HasSuffix(updateContext.PendingTargetName, "-"+targetId) {
					return fmt.Errorf("pending target %s does not match requested version %s", updateContext.PendingTargetName, targetId)
				}
			} else {
				// targetId is a target name, must match exactly
				log.Debug().Msg("targetId is a name, checking if PendingTargetName matches")
				if updateContext.PendingTargetName != targetId {
					return fmt.Errorf("pending target %s does not match requested target %s", updateContext.PendingTargetName, targetId)
				}
			}
		}

		if !updateContext.opts.DoCheck {
			// Check valid states for standalone fetch operation
			if updateState != update.StateInitialized && updateState != update.StateFetching {
				return fmt.Errorf("cannot fetch, current update is in '%s' state", updateState.String())
			}
		}
	}

	return nil
}

func Update(config *sotatoml.AppConfig, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		DbFilePath: path.Join(config.GetDefault("storage.path", "/var/sota"), config.GetDefault("storage.sqldb_path", "sql.db")),
	}

	var err error
	updateContext.Context = context.Background()
	updateContext.ComposeConfig, err = getComposeConfig(config)
	updateContext.opts = opts
	if err != nil {
		return err
	}

	err = GetPendingUpdate(updateContext)
	if err != nil {
		log.Err(err).Msg("Error getting pending update")
		return fmt.Errorf("error getting pending update: %w", err)
	}

	err = checkUpdateState(updateContext, opts.TargetId)
	if err != nil {
		return err
	}

	if updateContext.PendingTargetName != "" && (opts.DoInstall || opts.DoStart) {
		log.Info().Msgf("Proceeding with update to target %s", updateContext.PendingTargetName)
	}

	var localRepoPath string
	if opts.SrcDir == "" {
		localRepoPath = ""
	} else {
		localRepoPath = path.Join(opts.SrcDir, "repo")
	}

	client, err := transport.CreateClient(config)
	if err != nil {
		log.Err(err).Msg("Error creating HTTP client")
		return err
	}

	err = InitializeDatabase(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error initializing database")
		return err
	}

	updateContext.CurrentTarget, err = targets.GetCurrentTarget(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error getting current target")
	}

	updateContext.InstalledApps, updateContext.InstalledAppsNames, err = getInstalledApps(updateContext)
	if err != nil {
		log.Err(err).Msg("Error getting current apps")
	}

	var tufTargets map[string]*metadata.TargetFiles
	tufTargets, err = getTargets(config, localRepoPath, client, updateContext.CurrentTarget.Path, updateContext.InstalledAppsNames, opts.EnableTuf, opts.DoCheck)
	if err != nil {
		log.Err(err).Msg("Error getting targets")
		return err
	}

	var targetId string
	if opts.DoInstall || opts.DoStart {
		if !opts.DoFetch && updateContext.PendingTargetName == "" {
			log.Info().Msg("No pending target to update")
			return fmt.Errorf("no pending target to update")
		}
		if updateContext.PendingTargetName != "" {
			targetId = updateContext.PendingTargetName
			log.Debug().Msgf("Using pending target %s", updateContext.PendingTargetName)
		}
	}
	if targetId == "" {
		targetId = opts.TargetId
	}

	err = GetTargetToInstall(updateContext, config, tufTargets, targetId)
	if err != nil {
		return fmt.Errorf("error getting target to install %w", err)
	}

	if opts.DoCheck && !opts.DoFetch {
		// Log targets info when running standalone check command
		dumpTargetsInfo(tufTargets, updateContext)
	}

	if opts.DoFetch || opts.DoInstall || opts.DoStart {
		doRollback, err := PerformUpdate(updateContext)
		if doRollback {
			log.Err(err).Msgf("Error during update to target %s, rolling back", updateContext.Target.Path)
			rollbackErr := rollback(updateContext)
			if rollbackErr != nil {
				log.Err(rollbackErr).Msgf("Error rolling back")
				return fmt.Errorf("error rolling back: %w", rollbackErr)
			}
		} else {
			if err != nil {
				log.Err(err).Msgf("Error updating to target %s", updateContext.Target.Path)
			}
		}
	}

	_ = ReportAppsStates(config, client, updateContext)

	eventsUrl := config.GetDefault("tls.server", "https://ota-lite.foundries.io:8443") + "/events"
	errFlush := events.FlushEvents(updateContext.DbFilePath, client, eventsUrl)
	if errFlush != nil {
		log.Err(errFlush).Msg("Error flushing events")
	}
	return err
}

func dumpTargetsInfo(tufTargets map[string]*metadata.TargetFiles, updateContext *UpdateContext) {
	log.Info().Msgf("Available targets:")

	// Sort targets by version
	type targetInfo struct {
		Name    string
		Version int
	}
	var targetsList []targetInfo
	for name, t := range tufTargets {
		version, err := GetVersion(t)
		if err != nil {
			log.Err(err).Msgf("Error getting version for target %s", name)
			continue
		}
		targetsList = append(targetsList, targetInfo{Name: name, Version: version})
	}
	sort.Slice(targetsList, func(i, j int) bool {
		return targetsList[i].Version < targetsList[j].Version
	})

	// Print sorted list of targets
	for _, ti := range targetsList {
		log.Info().Msgf("  %s (version: %d)", ti.Name, ti.Version)
		apps, err := GetAppsUris(tufTargets[ti.Name])
		if err != nil {
			log.Err(err).Msgf("Error getting apps uris for target %s", ti.Name)
			continue
		}
		if len(apps) > 0 {
			log.Info().Msgf("    apps:")
			for _, app := range apps {
				log.Info().Msgf("      %s -> %s", getAppNameFromUri(app), app)
			}
		}
		log.Info().Msg("")
	}

	if updateContext.Target.Path == updateContext.CurrentTarget.Path {
		if updateContext.TargetIsRunning {
			if len(updateContext.AppsToUninstall) == 0 {
				log.Info().Msgf("Selected Target %s is already running", updateContext.Target.Path)
			} else {
				log.Info().Msgf("Selected Target %s is already running, but some apps need to be stopped: %v", updateContext.Target.Path, updateContext.AppsToUninstall)
			}
		} else {
			log.Info().Msgf("Selected Target %s is already running, but some apps need to be started", updateContext.Target.Path)
		}
	} else {
		log.Info().Msgf("Target %s needs to be installed", updateContext.Target.Path)
	}
}

func getAppNameFromUri(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return ""
	}
	appNameWithTag := parts[len(parts)-1]
	appNameParts := strings.Split(appNameWithTag, "@")
	return appNameParts[0]
}

func FillAppsList(updateContext *UpdateContext) error {
	var requiredApps []string
	if updateContext.PendingApps == nil {
		targetApps, err := GetAppsUris(updateContext.Target)
		if err != nil {
			log.Err(err).Msg("Error getting apps uris")
			return fmt.Errorf("error getting apps uris: %w", err)
		}
		requiredApps = []string{}
		for _, appUri := range targetApps {
			appName := getAppNameFromUri(appUri)
			if appName == "" {
				log.Warn().Msgf("App URI %s does not contain a valid app name", appUri)
				continue
			}
			if updateContext.ConfiguredAppNames == nil || slices.Contains(updateContext.ConfiguredAppNames, appName) {
				requiredApps = append(requiredApps, appUri)
			}
		}
		log.Debug().Msgf("targetApps: %v", targetApps)
		log.Debug().Msgf("Using filtered target apps: %v", requiredApps)
	} else {
		requiredApps = updateContext.PendingApps
		log.Debug().Msgf("Using pending update apps: %v", requiredApps)
	}

	updateContext.RequiredApps = requiredApps
	log.Debug().Msgf("installedApps: %v", updateContext.InstalledApps)
	log.Debug().Msgf("requiredApps: %v", requiredApps)
	appsToUninstall := []string{}
	for _, app := range updateContext.InstalledApps {
		if !slices.Contains(updateContext.RequiredApps, app) {
			appsToUninstall = append(appsToUninstall, app)
		}
	}
	updateContext.AppsToUninstall = appsToUninstall
	return nil
}

func FillAndCheckAppsList(updateContext *UpdateContext) error {
	err := FillAppsList(updateContext)
	if err != nil {
		log.Err(err).Msg("Error filling apps list")
		return fmt.Errorf("error filling apps list: %w", err)
	}

	log.Debug().Msgf("Checking if candidate target %s is running", updateContext.Target.Path)
	isRunning, err := IsTargetRunning(updateContext)
	if err != nil {
		return fmt.Errorf("error checking target: %w", err)
	}

	updateContext.TargetIsRunning = isRunning
	if isRunning {
		log.Debug().Msg("Target is running")
		if len(updateContext.AppsToUninstall) == 0 {
			log.Debug().Msg("No apps to uninstall")
		} else {
			log.Debug().Msgf("Apps to uninstall: %v", updateContext.AppsToUninstall)
		}
	}
	return nil
}

// Returns information about the apps to install and to remove, as long as the corresponding target
// No update operation is performed at this point. Not even apps stopping
func GetTargetToInstall(updateContext *UpdateContext, config *sotatoml.AppConfig, tufTargets map[string]*metadata.TargetFiles, targetId string) error {
	var err error

	specificVersion := -1
	specificName := ""
	if targetId != "" {
		if specificVersion, err = strconv.Atoi(targetId); err != nil {
			specificName = targetId
			specificVersion = -1
		}
	}

	candidateTarget, _ := selectTarget(tufTargets, specificVersion, specificName)
	if candidateTarget == nil {
		log.Info().Msgf("No target found for version %d", specificVersion)
		return fmt.Errorf("no target found for version %d", specificVersion)
	}

	if targetId == "" {
		// If no target is specified, check if automatically selected target is marked as failing
		failing, _ := targets.IsFailingTarget(updateContext.DbFilePath, candidateTarget.Path)
		if failing {
			log.Info().Msg("Skipping failing target " + candidateTarget.Path + " using " + updateContext.CurrentTarget.Path + " instead")
			candidateTarget = updateContext.CurrentTarget
		}
	}

	updateContext.Target = candidateTarget

	apps := config.GetDefault("pacman.compose_apps", "-")
	if apps != "-" {
		updateContext.ConfiguredAppNames = strings.Split(apps, ",")
		log.Debug().Msgf("pacman.compose_apps=%v", updateContext.ConfiguredAppNames)
	}

	err = FillAndCheckAppsList(updateContext)
	if err != nil {
		log.Err(err).Msg("FillAndCheckAppsList error")
		return err
	}

	if !updateContext.TargetIsRunning {
		if updateContext.CurrentTarget.Path != updateContext.Target.Path {
			updateContext.Reason = "Updating from " + updateContext.CurrentTarget.Path + " to " + updateContext.Target.Path
		} else {
			updateContext.Reason = "Syncing Active Target Apps"
		}
		log.Debug().Msg("Reason: " + updateContext.Reason)
	}
	return nil
}

func PerformUpdate(updateContext *UpdateContext) (bool, error) {
	// updateContext.Target must be set
	// updateContext.AppsToInstall might be empty
	if updateContext.Target.Path == updateContext.CurrentTarget.Path {
		if updateContext.TargetIsRunning {
			log.Info().Msgf("Target %s is already running", updateContext.Target.Path)
			if len(updateContext.AppsToUninstall) == 0 {
				log.Debug().Msgf("No apps to uninstall for target %s", updateContext.Target.Path)
				if updateContext.opts.DoFetch && updateContext.opts.TargetId == "" {
					return false, nil
				}
			} else {
				log.Info().Msgf("Uninstalling apps for target %s: %v", updateContext.Target.Path, updateContext.AppsToUninstall)
			}
		} else {
			log.Info().Msgf("Target %s is already running, but some apps need to be started", updateContext.Target.Path)
		}
	} else {
		log.Info().Msgf("%s", updateContext.Reason)
	}

	err := InitUpdate(updateContext)
	if err != nil {
		return false, fmt.Errorf("error initializing update for target: %w", err)
	}

	log.Debug().Msgf("updateContext.opts.DoPull: %v, updateContext.opts.DoInstall: %v, updateContext.opts.DoRun: %v", updateContext.opts.DoFetch, updateContext.opts.DoInstall, updateContext.opts.DoStart)
	// Fetch
	if updateContext.opts.DoFetch {
		err = PullTarget(updateContext)
		if err != nil {
			return false, fmt.Errorf("error pulling target: %w", err)
		}
	}

	// Install
	if updateContext.opts.DoInstall {
		err = InstallTarget(updateContext)
		if err != nil {
			return false, fmt.Errorf("error installing target: %w", err)
		}
	}

	// Start
	if updateContext.opts.DoStart {
		doRollback, err := StartTarget(updateContext)
		if err != nil {
			return doRollback, err
		}
	}
	return false, nil
}

func GenAndSaveEvent(updateContext *UpdateContext, eventType events.EventTypeValue, details string, success *bool) error {
	version, _ := GetVersion(updateContext.Target)
	targetName := updateContext.Target.Path
	evt := events.NewEvent(eventType, details, success, updateContext.Runner.Status().ID, targetName, version)
	return events.SaveEvent(updateContext.DbFilePath, &evt[0])
}

func GetAppsUris(target *metadata.TargetFiles) ([]string, error) {
	var tc targets.TargetCustom
	var b []byte
	b, _ = (*target.Custom).MarshalJSON()
	err := json.Unmarshal(b, &tc)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling event: %w", err)
	}

	var appsUris []string
	var dockerComposeApps map[string]interface{}
	err = json.Unmarshal(b, &dockerComposeApps)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling docker_compose_apps: %w", err)
	}

	if apps, ok := dockerComposeApps["docker_compose_apps"].(map[string]interface{}); ok {
		for _, app := range apps {
			if appDetails, ok := app.(map[string]interface{}); ok {
				if uri, ok := appDetails["uri"].(string); ok {
					appsUris = append(appsUris, uri)
				}
			}
		}
	} else {
		appsUris = []string{}
		// return nil, fmt.Errorf("docker_compose_apps field is missing or invalid")
	}

	return appsUris, nil
}

func GetVersion(target *metadata.TargetFiles) (int, error) {
	var tc targets.TargetCustom
	var b []byte
	b, _ = (*target.Custom).MarshalJSON()
	err := json.Unmarshal(b, &tc)
	if err != nil {
		return -1, fmt.Errorf("error unmarshaling event: %w", err)
	}
	version, err := strconv.Atoi(tc.Version)
	if err != nil {
		return -1, fmt.Errorf("error converting version to int: %w", err)
	}
	return version, nil
}

func selectTarget(allTargets map[string]*metadata.TargetFiles, specificVersion int, specificName string) (*metadata.TargetFiles, error) {
	log.Debug().Msgf("selectTarget: specificVersion=%d, specificName=%s", specificVersion, specificName)
	latest := -1
	var selectedTarget *metadata.TargetFiles
	for targetName := range allTargets {
		if specificName != "" && targetName == specificName {
			selectedTarget = allTargets[targetName]
			break
		}

		var tc targets.TargetCustom
		var b []byte
		b, _ = (*allTargets[targetName].Custom).MarshalJSON()
		err := json.Unmarshal(b, &tc)
		if err != nil {
			continue
		}

		v, err := strconv.Atoi(tc.Version)
		if err != nil {
			continue
		}
		if (specificVersion > 0 && specificVersion == v) || (specificVersion <= 0 && v > latest) {
			selectedTarget = allTargets[targetName]
			latest = v
			if specificVersion > 0 {
				break
			}
		}
	}
	return selectedTarget, nil
}

func getComposeConfig(config *sotatoml.AppConfig) (*compose.Config, error) {
	cfg, err := v1.NewDefaultConfig(
		v1.WithStoreRoot(config.GetDefault("pacman.reset_apps_root", "/var/sota/reset-apps")),
		v1.WithComposeRoot(config.GetDefault("pacman.compose_apps_root", "/var/sota/compose-apps")),
		v1.WithUpdateDB(path.Join(config.GetDefault("storage.path", "/var/sota"), "updates.db")),
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func CancelPendingUpdate(config *sotatoml.AppConfig, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		DbFilePath: path.Join(config.GetDefault("storage.path", "/var/sota"), config.GetDefault("storage.sqldb_path", "sql.db")),
	}

	var err error
	updateContext.Context = context.Background()
	updateContext.ComposeConfig, err = getComposeConfig(config)
	updateContext.opts = opts
	if err != nil {
		return err
	}

	err = GetPendingUpdate(updateContext)
	if err != nil {
		log.Err(err).Msg("Error getting pending update")
		return fmt.Errorf("error getting pending update: %w", err)
	}

	if updateContext.PendingRunner != nil {
		log.Info().Msgf("Canceling pending update to target %s", updateContext.PendingTargetName)
		err := updateContext.PendingRunner.Cancel(updateContext.Context)
		if err != nil {
			log.Err(err).Msg("Error canceling pending update")
			return fmt.Errorf("error canceling pending update: %w", err)
		}
	} else {
		log.Info().Msg("No pending update to cancel")
	}
	return nil
}

func Daemon(config *sotatoml.AppConfig, opts *UpdateOptions) {
	intervalStr := config.GetDefault("uptane.polling_seconds", "60")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		log.Err(err).Msgf("Invalid interval %s, using default 60 seconds", intervalStr)
		interval = 60
	}
	for {
		opts.DoCheck = true
		opts.DoFetch = true
		opts.DoInstall = true
		opts.DoStart = true
		err := Update(config, opts)
		if err != nil {
			log.Err(err).Msg("Error during update")
		}
		log.Info().Msgf("Waiting %d seconds before next update check", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func Status(config *sotatoml.AppConfig, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		DbFilePath: path.Join(config.GetDefault("storage.path", "/var/sota"), config.GetDefault("storage.sqldb_path", "sql.db")),
	}

	var err error
	updateContext.Context = context.Background()
	updateContext.ComposeConfig, err = getComposeConfig(config)
	updateContext.opts = opts
	if err != nil {
		return err
	}

	err = InitializeDatabase(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error initializing database")
		return err
	}

	err = GetPendingUpdate(updateContext)
	if err != nil {
		log.Err(err).Msg("Error getting pending update")
		return fmt.Errorf("error getting pending update: %w", err)
	}

	target, err := targets.GetCurrentTarget(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error getting current target")
		return fmt.Errorf("error getting current target: %w", err)
	}

	var currentUpdateURIs []string
	var lastSuccessfulUpdate *update.Update
	if lastSuccessfulUpdate, err = update.GetLastSuccessfulUpdate(updateContext.ComposeConfig); err == nil {
		currentUpdateURIs = lastSuccessfulUpdate.URIs
	}
	appStatuses, err := compose.CheckAppsStatus(updateContext.Context, updateContext.ComposeConfig, currentUpdateURIs)
	if err != nil {
		return fmt.Errorf("error checking apps status: %w", err)
	}

	fmt.Printf("Current: %s\n", target.Path)
	fmt.Printf("\tID:\t\t%s\n", lastSuccessfulUpdate.ID)
	fmt.Printf("\tCompleted:\t%s\n", lastSuccessfulUpdate.UpdateTime.Local().Format(time.DateTime))
	fmt.Println("\tApps:")
	printApps(lastSuccessfulUpdate.URIs)
	fmt.Println("\tStatus:")
	fmt.Printf("\t  Fetched:\t")
	if appStatuses.AreFetched() {
		fmt.Printf("Yes\n")
	} else {
		fmt.Printf("No\n")
	}
	fmt.Printf("\t  Installed:\t")
	if appStatuses.AreInstalled() {
		fmt.Printf("Yes\n")
	} else {
		fmt.Printf("No\n")
	}
	fmt.Printf("\t  Running:\t")
	if appStatuses.AreRunning() {
		fmt.Printf("Yes\n")
	} else {
		fmt.Printf("No\n")
	}

	var updateStatus *update.Update
	var updateType string
	var targetName string
	if updateContext.PendingRunner != nil {
		upd := updateContext.PendingRunner.Status()
		updateStatus = &upd
		updateType = "Ongoing"
	} else {
		updateStatus, err = update.GetFinalizedUpdate(updateContext.ComposeConfig)
		if err != nil {
			return fmt.Errorf("error getting last finalized update: %w", err)
		}
		updateType = "Last update"
	}
	targetName = updateStatus.ClientRef

	fmt.Printf("\n%s: %s\n", updateType, targetName)
	fmt.Printf("\tID:\t\t%s\n", updateStatus.ID)
	fmt.Printf("\tStarted:\t%s\n", updateStatus.CreationTime.Local().Format(time.DateTime))

	var updateTimeTitle string
	switch updateStatus.State {
	case update.StateCompleted:
		updateTimeTitle = "Completed"
	case update.StateFailed:
		updateTimeTitle = "Failed"
	case update.StateCanceled:
		updateTimeTitle = "Canceled"
	default:
		updateTimeTitle = "Last modified"
	}
	fmt.Printf("\t%s:\t%s\n", updateTimeTitle, updateStatus.UpdateTime.Local().Format(time.DateTime))
	fmt.Println("\tApps:")
	printApps(updateStatus.URIs)
	fmt.Printf("\tSize: \t\t%s; %d blobs\n", units.BytesSize(float64(updateStatus.TotalBlobsBytes)), len(updateStatus.Blobs))

	fmt.Printf("\tState: \t\t%s\n", updateStatus.State)
	if updateStatus.State == update.StateFetching {
		fmt.Printf("\t\tProgress: %d%%\n", updateStatus.Progress)
		fmt.Printf("\t\tBytes: %s; %d blobs\n", units.BytesSize(float64(updateStatus.FetchedBytes)), updateStatus.FetchedBlobs)
	}

	return nil
}

func printApps(appURIs []string) {
	for _, appURI := range appURIs {
		appName := "n/a"
		if ref, err := compose.ParseAppRef(appURI); err == nil {
			appName = ref.Name
		} else {
			log.Warn().Msgf("error parsing app ref %s: %v", appURI, err)
		}
		fmt.Printf("\t\t\t%s\n", appName)
		fmt.Printf("\t\t\t  - %s\n", appURI)
	}
}
