// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/internal/targets"
	dg "github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	_ "modernc.org/sqlite"
)

// Atributes of the UpdateContext instance are gradually set during the update process
type (
	UpdateContext struct {
		opts       *UpdateOptions
		DbFilePath string

		Target             target.Target
		CurrentTarget      target.Target
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

func checkUpdateState(updateContext *UpdateContext, targetId string) error {
	slog.Debug("Checking update state", "target_id", targetId)
	// standalone check command has no state requirements
	if updateContext.opts.DoCheck && !updateContext.opts.DoFetch {
		slog.Debug("Standalone check command, no state requirements")
		return nil
	}

	var updateState update.State
	if updateContext.PendingRunner != nil {
		updateState = updateContext.PendingRunner.Status().State
	}

	// standalone install and start commands require a pending update operation at the right state
	if (updateContext.opts.DoInstall || updateContext.opts.DoStart) && !updateContext.opts.DoFetch {
		slog.Debug("Standalone install or start command, checking requirements")
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
		slog.Debug("Update or standalone fetch command, checking requirements")
		if targetId != "" {
			if _, err := strconv.Atoi(targetId); err == nil {
				// targetId is a version, check if PendingTargetName ends with -<version>
				slog.Debug("targetId is a version, checking if PendingTargetName ends with -<version>")
				if !strings.HasSuffix(updateContext.PendingTargetName, "-"+targetId) {
					return fmt.Errorf("pending target %s does not match requested version %s", updateContext.PendingTargetName, targetId)
				}
			} else {
				// targetId is a target name, must match exactly
				slog.Debug("targetId is a name, checking if PendingTargetName matches")
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

func Update(ctx context.Context, cfg *config.Config, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		Context:       ctx,
		DbFilePath:    cfg.GetDBPath(),
		ComposeConfig: cfg.ComposeConfig(),
		opts:          opts,
	}

	err := GetPendingUpdate(updateContext)
	if err != nil {
		return fmt.Errorf("error getting pending update: %w", err)
	}

	err = checkUpdateState(updateContext, opts.TargetId)
	if err != nil {
		return err
	}

	if updateContext.PendingTargetName != "" && (opts.DoInstall || opts.DoStart) {
		slog.Info("Proceeding with update to target", "target", updateContext.PendingTargetName)
	}

	err = InitializeDatabase(updateContext.DbFilePath)
	if err != nil {
		return fmt.Errorf("error initializing database: %w", err)
	}

	updateContext.CurrentTarget, err = targets.GetCurrentTarget(updateContext.DbFilePath)
	if err != nil {
		slog.Error("Error getting current target", "error", err)
	}

	updateContext.InstalledApps, updateContext.InstalledAppsNames, err = getInstalledApps(updateContext)
	if err != nil {
		slog.Error("Error getting current apps", "error", err)
	}

	var client *dg.GatewayClient
	var targetsRepo target.Repo
	client, err = dg.NewGatewayClient(cfg, updateContext.CurrentTarget.AppNames(), updateContext.CurrentTarget.ID)
	if err != nil {
		return err
	}
	if opts.EnableTuf {
		targetsRepo, err = target.NewTufRepo(cfg, client, cfg.GetHardwareID())
	} else {
		targetsRepo, err = target.NewPlainRepo(client, cfg.GetTargetsFilepath(), cfg.GetHardwareID())
	}
	if err != nil {
		return err
	}
	var targetList target.Targets
	targetList, err = targetsRepo.LoadTargets(opts.DoCheck)
	if err != nil {
		return err
	}
	var targetId string
	if opts.DoInstall || opts.DoStart {
		if !opts.DoFetch && updateContext.PendingTargetName == "" {
			slog.Info("No pending target to update")
			return fmt.Errorf("no pending target to update")
		}
		if updateContext.PendingTargetName != "" {
			targetId = updateContext.PendingTargetName
			slog.Debug("Using pending target", "target_id", updateContext.PendingTargetName)
		}
	}
	if targetId == "" {
		targetId = opts.TargetId
	}

	err = GetTargetToInstall(updateContext, cfg, targetList, targetId)
	if err != nil {
		return fmt.Errorf("error getting target to install %w", err)
	}

	if opts.DoCheck && !opts.DoFetch {
		// Log targetList info when running standalone check command
		dumpTargetsInfo(targetList, updateContext)
	}

	if opts.DoFetch || opts.DoInstall || opts.DoStart {
		doRollback, err := PerformUpdate(updateContext)
		if doRollback {
			slog.Error("Error during update, rolling back", "target_id", updateContext.Target.ID, "error", err)
			rollbackErr := rollback(updateContext)
			if rollbackErr != nil {
				return fmt.Errorf("error rolling back: %w", rollbackErr)
			}
		} else {
			if err != nil {
				slog.Error("Error updating to target", "target_id", updateContext.Target.ID, "error", err)
			}
		}
	}

	if err == nil {
		// If successful update then update headers of the gateway client so it knows about
		// the current target name and apps
		client.UpdateHeaders(updateContext.Target.AppNames(), updateContext.Target.ID)
	}
	_ = ReportAppsStates(cfg.TomlConfig(), client, updateContext)

	errFlush := events.FlushEvents(updateContext.DbFilePath, client)
	if errFlush != nil {
		slog.Error("Error flushing events", "error", errFlush)
	}
	return err
}

func dumpTargetsInfo(targets target.Targets, updateContext *UpdateContext) {
	fmt.Println("Available targets:")

	targetsList := targets.GetSortedList()

	// Print sorted list of targets
	for _, ti := range targetsList {
		fmt.Printf("  %s (version: %d)\n", ti.ID, ti.Version)
		if len(ti.Apps) > 0 {
			for _, app := range ti.Apps {
				fmt.Printf("      %s -> %s\n", app.Name, app.URI)
			}
		}
		fmt.Println("")
	}

	if updateContext.Target.ID == updateContext.CurrentTarget.ID {
		if updateContext.TargetIsRunning {
			if len(updateContext.AppsToUninstall) == 0 {
				fmt.Printf("Selected Target %s is already running\n", updateContext.Target.ID)
			} else {
				fmt.Printf("Selected Target %s is already running, but some apps need to be stopped: %v\n", updateContext.Target.ID, updateContext.AppsToUninstall)
			}
		} else {
			fmt.Printf("Selected Target %s is already running, but some apps need to be started\n", updateContext.Target.ID)
		}
	} else {
		fmt.Printf("Target %s needs to be installed\n", updateContext.Target.ID)
	}
}

func FillAppsList(updateContext *UpdateContext) error {
	var requiredApps []string
	if updateContext.PendingApps == nil {
		targetApps := updateContext.Target.Apps
		requiredApps = []string{}
		for _, appUri := range targetApps {
			appName := appUri.Name
			if appName == "" {
				slog.Warn("App URI does not contain a valid app name", "app_uri", appUri)
				continue
			}
			if updateContext.ConfiguredAppNames == nil || slices.Contains(updateContext.ConfiguredAppNames, appName) {
				requiredApps = append(requiredApps, appUri.URI)
			}
		}
		slog.Debug("Using filtered target apps", "required_apps", requiredApps, "target_apps", targetApps, "installed_apps", updateContext.InstalledApps)
	} else {
		requiredApps = updateContext.PendingApps
		slog.Debug("Using pending update apps", "required_apps", requiredApps, "installed_apps", updateContext.InstalledApps)
	}

	updateContext.RequiredApps = requiredApps
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
		return fmt.Errorf("error filling apps list: %w", err)
	}

	slog.Debug("Checking if candidate target is running", "target_id", updateContext.Target.ID)
	isRunning, err := IsTargetRunning(updateContext)
	if err != nil {
		return fmt.Errorf("error checking target: %w", err)
	}

	updateContext.TargetIsRunning = isRunning
	if isRunning {
		slog.Debug("Target is running")
		if len(updateContext.AppsToUninstall) == 0 {
			slog.Debug("No apps to uninstall")
		} else {
			slog.Debug("There are apps to uninstall", "apps_to_uninstall", updateContext.AppsToUninstall)
		}
	}
	return nil
}

// Returns information about the apps to install and to remove, as long as the corresponding target
// No update operation is performed at this point. Not even apps stopping
func GetTargetToInstall(updateContext *UpdateContext, cfg *config.Config, targetList target.Targets, targetId string) error {
	var err error
	specificVersion := -1
	specificName := ""
	if targetId != "" {
		if specificVersion, err = strconv.Atoi(targetId); err != nil {
			specificName = targetId
			specificVersion = -1
		}
	}

	candidateTarget, _ := selectTarget(targetList, specificVersion, specificName)
	if candidateTarget.ID == target.UnknownTarget.ID {
		return fmt.Errorf("no target found for version %d", specificVersion)
	}

	if targetId == "" {
		// If no target is specified, check if automatically selected target is marked as failing
		failing, _ := targets.IsFailingTarget(updateContext.DbFilePath, candidateTarget.ID)
		if failing {
			slog.Info("Skipping failing target " + candidateTarget.ID + " using " + updateContext.CurrentTarget.ID + " instead")
			candidateTarget = updateContext.CurrentTarget
		}
	}

	candidateTarget.ShortlistApps(cfg.GetEnabledApps())
	updateContext.Target = candidateTarget

	apps := cfg.GetEnabledApps()
	if apps != nil {
		updateContext.ConfiguredAppNames = apps
		slog.Debug("Selected compose apps", "apps_names", updateContext.ConfiguredAppNames)
	}

	err = FillAndCheckAppsList(updateContext)
	if err != nil {
		return fmt.Errorf("error filling and checking apps list: %w", err)
	}

	if !updateContext.TargetIsRunning {
		if updateContext.CurrentTarget.ID != updateContext.Target.ID {
			updateContext.Reason = "Updating from " + updateContext.CurrentTarget.ID + " to " + updateContext.Target.ID
		} else {
			updateContext.Reason = "Syncing Active Target Apps"
		}
		slog.Debug("Reason: " + updateContext.Reason)
	}
	return nil
}

func PerformUpdate(updateContext *UpdateContext) (bool, error) {
	// updateContext.Target must be set
	// updateContext.AppsToInstall might be empty
	if updateContext.Target.ID == updateContext.CurrentTarget.ID {
		if updateContext.TargetIsRunning {
			slog.Info("Target is already running", "target_id", updateContext.Target.ID)
			if len(updateContext.AppsToUninstall) == 0 {
				slog.Debug("No apps to uninstall", "target_id", updateContext.Target.ID)
				if updateContext.opts.DoFetch && updateContext.opts.TargetId == "" {
					return false, nil
				}
			} else {
				slog.Info("Uninstalling apps for target", "target_id", updateContext.Target.ID, "apps_to_uninstall", updateContext.AppsToUninstall)
			}
		} else {
			slog.Info("Target is already running, but some apps need to be started", "target_id", updateContext.Target.ID)
		}
	} else {
		slog.Info(updateContext.Reason)
	}

	err := InitUpdate(updateContext)
	if err != nil {
		return false, fmt.Errorf("error initializing update for target: %w", err)
	}

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
	evt := events.NewEvent(eventType, details, success, updateContext.Runner.Status().ID, updateContext.Target.ID, updateContext.Target.Version)
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

func selectTarget(targets target.Targets, specificVersion int, specificName string) (target.Target, error) {
	slog.Debug("Selecting target", "specific_version", specificVersion, "specific_name", specificName)

	if len(specificName) > 0 {
		return targets.GetTargetByID(specificName), nil
	} else if specificVersion != -1 {
		return targets.GetTargetByVersion(specificVersion), nil
	}
	return targets.GetLatestTarget(), nil
}

func CancelPendingUpdate(ctx context.Context, cfg *config.Config, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		Context:       ctx,
		ComposeConfig: cfg.ComposeConfig(),
		DbFilePath:    cfg.GetDBPath(),
		opts:          opts,
	}

	err := GetPendingUpdate(updateContext)
	if err != nil {
		return fmt.Errorf("error getting pending update: %w", err)
	}

	if updateContext.PendingRunner != nil {
		slog.Info("Canceling pending update to target", "target_id", updateContext.PendingTargetName)
		err := updateContext.PendingRunner.Cancel(updateContext.Context)
		if err != nil {
			return fmt.Errorf("error canceling pending update: %w", err)
		}
	} else {
		slog.Info("No pending update to cancel")
	}
	return nil
}

func Daemon(ctx context.Context, cfg *config.Config, opts *UpdateOptions) {
	intervalStr := cfg.TomlConfig().GetDefault("uptane.polling_seconds", "60")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		slog.Error("Invalid interval, using default 60 seconds", "interval", intervalStr, "error", err)
		interval = 60
	}
	for {
		opts.DoCheck = true
		opts.DoFetch = true
		opts.DoInstall = true
		opts.DoStart = true
		err := Update(ctx, cfg, opts)
		if err != nil {
			slog.Error("Error during update", "error", err)
		}
		slog.Info("Waiting before next update check", "interval", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
