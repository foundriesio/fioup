package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

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
		opts       UpdateOptions
		DbFilePath string

		Target             *metadata.TargetFiles
		CurrentTarget      *metadata.TargetFiles
		Reason             string
		RequiredApps       []string
		AppsToUninstall    []string
		InstalledApps      []string
		ConfiguredAppNames []string

		Context       context.Context
		ComposeConfig *compose.Config
		Runner        update.Runner
		Resuming      bool
		CorrelationId string
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

func getTargetsTuf(config *sotatoml.AppConfig, localRepoPath string, client *http.Client) (map[string]*metadata.TargetFiles, error) {
	fiotuf, err := tuf.NewFioTuf(config, client)
	if err != nil {
		log.Err(err).Msg("Error creating fiotuf instance")
		return nil, err
	}

	err = fiotuf.RefreshTuf(localRepoPath)
	if err != nil {
		log.Err(err).Msg("Error refreshing TUF")
		return nil, err
	}

	tufTargets := fiotuf.GetTargets()
	return tufTargets, nil
}

func fetchTargetsJson(config *sotatoml.AppConfig, client *http.Client) ([]byte, error) {
	urlPath := config.GetDefault("tls.server", "https://ota-lite.foundries.io:8443") + "/repo/targets.json"
	headers := make(map[string]string)
	headers["x-ats-tags"] = config.Get("pacman.tags")
	res, err := transport.HttpGet(client, urlPath, headers)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", res.StatusCode, urlPath)
	}
	return res.Body, nil
}

func getTargetsUnsafe(config *sotatoml.AppConfig, localRepoPath string, client *http.Client) (map[string]*metadata.TargetFiles, error) {
	var targetsBytes []byte
	var err error
	if localRepoPath == "" {
		targetsBytes, err = fetchTargetsJson(config, client)
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

	meta, err := metadata.Targets().FromBytes(targetsBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing targets.json: %w", err)
	}
	targets := meta.Signed.Targets

	return targets, nil

}

func getTargets(config *sotatoml.AppConfig, localRepoPath string, client *http.Client, enableTuf bool) (map[string]*metadata.TargetFiles, error) {
	if enableTuf {
		return getTargetsTuf(config, localRepoPath, client)
	} else {
		return getTargetsUnsafe(config, localRepoPath, client)
	}
}

// Runs check + update (if needed) once. May become a loop in the future
func Update(config *sotatoml.AppConfig, opts *UpdateOptions) error {
	updateContext := &UpdateContext{
		DbFilePath: path.Join(config.GetDefault("storage.path", "/var/sota"), config.GetDefault("storage.sqldb_path", "sql.db")),
	}
	err := InitializeDatabase(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error initializing database")
		return err
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

	tufTargets, err := getTargets(config, localRepoPath, client, opts.EnableTuf)
	if err != nil {
		log.Err(err).Msg("Error getting targets")
		return err
	}

	err = GetTargetToInstall(updateContext, config, tufTargets, opts.TargetId)
	if err != nil {
		return fmt.Errorf("error getting target to install %w", err)
	}

	_, err = PerformUpdate(updateContext)
	// if doRollback {
	// 	log.Info().Msg("Rolling back", err)
	// 	err = Rollback(updateContext)
	// 	if err != nil {
	// 		log.Info().Msg("Error rolling back", err)
	// 		return err
	// 	}
	// }
	if err != nil {
		log.Err(err).Msg("Error updating to target")
	}

	ReportAppsStates(config, client, updateContext)

	eventsUrl := config.GetDefault("tls.server", "https://ota-lite.foundries.io:8443") + "/events"
	events.FlushEvents(updateContext.DbFilePath, client, eventsUrl)
	return err
}

func ReportAppsStates(config *sotatoml.AppConfig, client *http.Client, updateContext *UpdateContext) error {
	log.Debug().Msg("Reporting apps state (stub)")

	// states, err := compose.CheckAppsStatus(updateContext.Context, updateContext.ComposeConfig, nil)
	// if err != nil {
	// 	log.Err(err).Msg("Error checking apps status")
	// 	return err
	// }

	currentTime := time.Now()
	utcTime := currentTime.UTC()
	rfc3339Time := utcTime.Format(time.RFC3339)

	data := map[string]interface{}{
		"deviceTime": rfc3339Time,
		"ostree":     "8509e5bda0c762d4bac7f90d79c2f9bf560f0cdac2c4a2d6361a041a5a677566",
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	log.Printf("Apps states: %s", string(dataBytes))
	appsStatesUrl := config.GetDefault("tls.server", "https://ota-lite.foundries.io:8443") + "/apps-states"

	res, err := transport.HttpPost(client, appsStatesUrl, data)
	if err != nil {
		log.Printf("Unable to send apps-state: %s", err)
	} else if res.StatusCode < 200 || res.StatusCode > 204 {
		log.Printf("Server could not process apps-states: HTTP_%d - %s", res.StatusCode, res.String())
	}
	return err
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
	targetApps, err := GetAppsUris(updateContext.Target)
	if err != nil {
		log.Err(err).Msg("Error getting apps uris")
		return fmt.Errorf("error getting apps uris: %w", err)
	}

	requiredApps := []string{}
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
	updateContext.RequiredApps = requiredApps

	installedApps, err := getInstalledApps(updateContext)
	log.Debug().Msg(fmt.Sprintf("targetApps: %v", targetApps))
	log.Debug().Msg(fmt.Sprintf("installedApps: %v", installedApps))
	log.Debug().Msg(fmt.Sprintf("requiredApps: %v", requiredApps))
	if err != nil {
		log.Err(err).Msg("Error getting running apps")
		return fmt.Errorf("error getting running apps: %w", err)
	}
	appsToUninstall := []string{}
	for _, app := range installedApps {
		if !slices.Contains(updateContext.RequiredApps, app) {
			appsToUninstall = append(appsToUninstall, app)
		}
	}
	updateContext.AppsToUninstall = appsToUninstall
	updateContext.InstalledApps = installedApps
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

	if isRunning {
		log.Debug().Msg("Target is running")
		updateContext.Target = nil
		updateContext.RequiredApps = nil
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

	updateContext.ComposeConfig, err = getComposeConfig(config)
	if err != nil {
		return err
	}

	currentTarget, err := targets.GetCurrentTarget(updateContext.DbFilePath)
	if err != nil {
		log.Err(err).Msg("Error getting current target")
	}

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
		// If no specific target is specified, check if automatically selected target is marked as failing
		failing, _ := targets.IsFailingTarget(updateContext.DbFilePath, candidateTarget.Path)
		if failing {
			log.Info().Msg("Skipping failing target " + candidateTarget.Path + " using " + currentTarget.Path + " instead")
			candidateTarget = currentTarget
		}
	}

	updateContext.Target = candidateTarget
	updateContext.CurrentTarget = currentTarget
	updateContext.Context = context.Background()

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

	// No update required
	if updateContext.Target == nil {
		log.Debug().Msg("No update required")
		return nil
	}

	if updateContext.CurrentTarget.Path != updateContext.Target.Path {
		updateContext.Reason = "Updating from " + updateContext.CurrentTarget.Path + " to " + updateContext.Target.Path
	} else {
		updateContext.Reason = "Syncing Active Target Apps"
	}
	log.Debug().Msg("Reason: " + updateContext.Reason)
	return nil
}

// Perform the actual update based on information collected before
func PerformUpdate(updateContext *UpdateContext) (bool, error) {
	// Valid cases:
	// If updateContext.Target is set, it is either an apps sync or version update. Events will be generated. updateContext.AppsToUninstall will be explicitly handled
	//   - If updateContext.AppsToInstall is empty, we will not initiate a composeapp update.
	//   - If updateContext.AppsToInstall is set, we will initiate a composeapp update.
	// If updateContext.Target is not set, updateContext.AppsToInstall shouldn't be set, and only handle updateContext.AppsToUninstall

	if updateContext.Target == nil {
		if len(updateContext.AppsToUninstall) == 0 {
			log.Info().Msgf("Target %s is already running, nothing to do", updateContext.CurrentTarget.Path)
			return false, nil
		} else {
			log.Info().Msgf("Target %s is already running, but the following apps need to be uninstalled: %v", updateContext.CurrentTarget.Path, updateContext.AppsToUninstall)
			return false, StopAndRemoveApps(updateContext)
		}

	} else {
		return UpdateToTarget(updateContext)
	}
}

func UpdateToTarget(updateContext *UpdateContext) (bool, error) {
	// updateContext.Target must be set
	// updateContext.AppsToInstall might be empty. In this case, we will not initiate a composeapp update, just remove the required apps and geenerate the events

	if updateContext.Target.Path == updateContext.CurrentTarget.Path {
		log.Info().Msgf("Target %s is already running, but some apps need to be started", updateContext.Target.Path)
	} else {
		log.Info().Msgf("%s", updateContext.Reason)
	}

	err := InitUpdate(updateContext)
	if err != nil {
		return false, fmt.Errorf("error initializing update for target: %w", err)
	}

	// Pull
	err = PullTarget(updateContext)
	if err != nil {
		return false, fmt.Errorf("error pulling target: %w", err)
	}

	// Install
	err = InstallTarget(updateContext)
	if err != nil {
		return false, fmt.Errorf("error installing target: %w", err)
	}

	// Run
	doRollback, err := StartTarget(updateContext)
	if err != nil {
		return doRollback, fmt.Errorf("error running target: %w", err)
	}
	log.Info().Msgf("Update complete")

	return false, nil
}

func GenAndSaveEvent(updateContext *UpdateContext, eventType events.EventTypeValue, details string, success *bool) error {
	version, _ := GetVersion(updateContext.Target)
	targetName := updateContext.Target.Path
	evt := events.NewEvent(eventType, details, success, updateContext.CorrelationId, targetName, version)
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
