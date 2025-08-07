package update

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/internal/targets"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	_ "modernc.org/sqlite"
)

type (
	UpdateOptions struct {
		SrcDir    string
		EnableTuf bool
		TargetId  string
	}
)

func InitUpdate(updateContext *UpdateContext) error {
	log.Info().Msgf("Initializing update for target %s", updateContext.Target.Path)
	updateRunner, err := update.GetCurrentUpdate(updateContext.ComposeConfig)
	var correlationId string
	if !errors.Is(err, update.ErrUpdateNotFound) {
		updateStatus := updateRunner.Status()
		log.Debug().Msgf("Current update: %v", updateStatus)

		clientRef := updateStatus.ClientRef
		clientRefSplit := strings.Split(clientRef, "|")
		if (clientRefSplit == nil) || (len(clientRefSplit) != 2) {
			log.Info().Msgf("Invalid clientRef: %s", clientRef)
			err = updateRunner.Cancel(updateContext.Context)
			if err != nil {
				return fmt.Errorf("error cancelling update: %w", err)
			}
		}

		targetName := clientRefSplit[0]
		correlationId = clientRefSplit[1]

		if updateStatus.State == update.StateStarted {
			updateRunner.Complete(updateContext.Context)
		}

		updateStatus = updateRunner.Status()
		if updateStatus.State != update.StateCompleted {
			if updateStatus.State != update.StateInitializing && updateStatus.State != update.StateCanceled && updateStatus.State != update.StateCancelling && targetName == updateContext.Target.Path && appsListMatch(updateContext.RequiredApps, updateStatus.URIs) {
				log.Info().Msgf("Proceeding with previous update of target %s", targetName)
				updateContext.Resuming = true
			} else {
				log.Debug().Msgf("Cancelling current update: %s", updateStatus.ID)
				correlationId = ""
				err = updateRunner.Cancel(updateContext.Context)
				if err != nil {
					return fmt.Errorf("error cancelling update: %w", err)
				}
			}
		}
	}

	if !updateContext.Resuming {
		version, err := GetVersion(updateContext.Target)
		if err != nil {
			return fmt.Errorf("error getting version: %w", err)
		}
		correlationId = fmt.Sprintf("%d-%d", version, time.Now().Unix())

		updateRunner, err = update.NewUpdate(updateContext.ComposeConfig, updateContext.Target.Path+"|"+correlationId)
		if err != nil {
			return err
		}

		// Progress bar
		bar := progressbar.DefaultBytes(int64(len(updateContext.RequiredApps)) + 1)
		initOptions := []update.InitOption{
			update.WithInitProgress(func(status *update.InitProgress) {
				if status.Current == 0 {
					return
				}
				if status.State == update.UpdateInitStateLoadingTree {
					if err := bar.Set(status.Current + 1); err != nil {
						log.Err(err).Msg("Error setting progress bar")
					}
				} else {
					if bar == nil {
						bar = progressbar.Default(int64(status.Total))
					}
					if err := bar.Set(status.Current + 1); err != nil {
						log.Err(err).Msg("Error setting progress bar")
					}
				}
			}), update.WithInitAllowEmptyAppList(true), update.WithInitCheckStatus(false)}

		err = updateRunner.Init(updateContext.Context, updateContext.RequiredApps, initOptions...)
		if err != nil {
			return err
		}
	}
	updateContext.Runner = updateRunner
	updateContext.CorrelationId = correlationId
	return nil
}

func PullTarget(updateContext *UpdateContext) error {
	log.Info().Msgf("Pulling target %v", updateContext.Target.Path)

	var updateStatus update.Update
	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateInitialized && updateStatus.State != update.StateFetching {
		log.Info().Msgf("update has already been fetched. Update state: %s", updateStatus.State)
		if updateContext.Resuming {
			return nil
		}
	}

	err := GenAndSaveEvent(updateContext, events.DownloadStarted, updateContext.Reason, nil)
	if err != nil {
		return fmt.Errorf("error on GenAndSaveEvent: %w", err)
	}

	// Progress bar
	bar := progressbar.DefaultBytes(updateStatus.TotalBlobsBytes + 1)
	fetchOptions := []compose.FetchOption{
		compose.WithFetchProgress(func(status *compose.FetchProgress) {
			if err := bar.Set64(status.CurrentBytes + 1); err != nil {
				log.Err(err).Msg("Error setting progress bar")
			}
		}),
		compose.WithProgressPollInterval(200)}

	err = updateContext.Runner.Fetch(updateContext.Context, fetchOptions...)
	if err != nil {
		GenAndSaveEvent(updateContext, events.DownloadCompleted, err.Error(), targets.BoolPointer(false))
		return fmt.Errorf("error pulling target: %w", err)
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateFetched {
		log.Info().Msg("update not fetched")
	}
	if updateStatus.Progress != 100 {
		log.Info().Msgf("update is not fetched for 100%%: %d", updateStatus.Progress)
	}

	err = GenAndSaveEvent(updateContext, events.DownloadCompleted, "", targets.BoolPointer(true))
	if err != nil {
		return fmt.Errorf("error on GenAndSaveEvent: %w", err)
	}

	return nil
}

type progressRendererCtx struct {
	bar        *progressbar.ProgressBar
	curImageID string
	curLayerID string
}

func getProgressRenderer() compose.InstallProgressFunc {
	ctx := &progressRendererCtx{}

	return func(p *compose.InstallProgress) {
		switch p.AppInstallState {
		case compose.AppInstallStateComposeInstalling:
			{
				log.Info().Msgf("Installing app %s", p.AppID)
			}
		case compose.AppInstallStateComposeChecking:
			{
			}
		case compose.AppInstallStateImagesLoading:
			{
				renderImageLoadingProgress(ctx, p)
			}
		}
	}
}

func renderImageLoadingProgress(ctx *progressRendererCtx, p *compose.InstallProgress) {
	switch p.ImageLoadState {
	case compose.ImageLoadStateLayerLoading:
		{
			if ctx.curImageID != p.ImageID {
				log.Printf("  Loading image %s", p.ImageID)
				ctx.curImageID = p.ImageID
				ctx.curLayerID = ""
			}
			if ctx.curLayerID != p.ID {
				ctx.bar = progressbar.DefaultBytes(p.Total)
				ctx.bar.Describe(fmt.Sprintf("    %s", p.ID))
				ctx.curLayerID = p.ID
			}
			if err := ctx.bar.Set64(p.Current); err != nil {
				log.Printf("Error setting progress bar: %s", err.Error())
			}
		}
	case compose.ImageLoadStateLayerSyncing:
		{
			// TODO: render layer syncing progress
			//fmt.Print(".")
		}
	case compose.ImageLoadStateLayerLoaded:
		{
			//fmt.Println("ok")
			ctx.curLayerID = ""
			ctx.bar.Close()
			ctx.bar = nil
		}
	case compose.ImageLoadStateImageLoaded:
		{
			log.Debug().Msgf("  Image loaded: %s", p.ImageID)
		}
	case compose.ImageLoadStateImageExist:
		{
			log.Debug().Msgf("  Already exists: %s", p.ImageID)
		}
	default:
		log.Debug().Msgf("  Unknown state %s", p.ImageLoadState)
	}
}

func InstallTarget(updateContext *UpdateContext) error {
	log.Info().Msgf("Installing target %v", updateContext.Target.Path)

	updateStatus := updateContext.Runner.Status()
	if updateStatus.State != update.StateFetched && updateStatus.State != update.StateInstalling {
		log.Debug().Msgf("update was already installed. Update state: %s", updateStatus.State)
		if updateContext.Resuming {
			return nil
		}
	}

	targets.RegisterInstallationStarted(updateContext.DbFilePath, updateContext.Target, updateContext.CorrelationId)
	err := GenAndSaveEvent(updateContext, events.InstallationStarted, updateContext.Reason, nil)
	if err != nil {
		log.Err(err).Msg("error on GenAndSaveEvent")
	}

	installOptions := []compose.InstallOption{
		compose.WithInstallProgress(getProgressRenderer())}

	compose.StopApps(updateContext.Context, updateContext.ComposeConfig, updateContext.AppsToUninstall)
	err = updateContext.Runner.Install(updateContext.Context, installOptions...)
	if err != nil {
		err := GenAndSaveEvent(updateContext, events.DownloadCompleted, err.Error(), targets.BoolPointer(false))
		return fmt.Errorf("error installing target: %w", err)
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateInstalled {
		log.Debug().Msg("update not installed")
	}
	if updateStatus.Progress != 100 {
		log.Debug().Msgf("update is not installed for 100%%: %d", updateStatus.Progress)
	}

	err = GenAndSaveEvent(updateContext, events.InstallationApplied, "", targets.BoolPointer(true))
	if err != nil {
		log.Err(err).Msg("error on GenAndSaveEvent")
	}
	return nil
}

func StartTarget(updateContext *UpdateContext) (bool, error) {
	log.Info().Msgf("Running target %v", updateContext.Target.Path)

	var err error
	updateStatus := updateContext.Runner.Status()
	if updateStatus.State != update.StateInstalled && updateStatus.State != update.StateStarting {
		log.Debug().Msgf("Skipping start target operation because state is: %s", updateStatus.State)
		if updateContext.Resuming {
			return false, nil
		}
	}

	compose.StopApps(updateContext.Context, updateContext.ComposeConfig, updateContext.AppsToUninstall)

	err = updateContext.Runner.Start(updateContext.Context)
	if err != nil {
		log.Err(err).Msg("error on starting target")
		err := GenAndSaveEvent(updateContext, events.InstallationCompleted, err.Error(), targets.BoolPointer(false))
		if err != nil {
			log.Err(err).Msg("error on GenAndSaveEvent")
		}
		targets.RegisterInstallationFailed(updateContext.DbFilePath, updateContext.Target, updateContext.CorrelationId)

		rollback(updateContext)

		return false, fmt.Errorf("rolled back to previous target")
	}

	if updateContext.Runner.Status().State != update.StateStarted {
		log.Info().Msg("update not started")
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.Progress != 100 {
		log.Debug().Msgf("update is not started for 100%%: %d", updateStatus.Progress)
	}

	err = GenAndSaveEvent(updateContext, events.InstallationCompleted, "", targets.BoolPointer(true))
	if err != nil {
		log.Err(err).Msg("error on GenAndSaveEvent")
	}
	targets.RegisterInstallationSuceeded(updateContext.DbFilePath, updateContext.Target, updateContext.CorrelationId)

	log.Debug().Msg("Completing update with pruning")
	err = updateContext.Runner.Complete(updateContext.Context, update.CompleteWithPruning())
	if err != nil {
		log.Err(err).Msg("error completing update:")
	}

	return false, nil
}

func rollback(updateContext *UpdateContext) error {
	log.Info().Msgf("Rolling back to target %v", updateContext.CurrentTarget.Path)

	if updateContext.Runner != nil {

		updateStatus := updateContext.Runner.Status()
		if updateStatus.State == update.StateStarted {
			err := updateContext.Runner.Complete(updateContext.Context)
			if err != nil {
				log.Err(err).Msg("Rollback: Error updateContext.Runner.Complete")
			}
		} else {
			err := updateContext.Runner.Cancel(updateContext.Context)
			if err != nil {
				log.Err(err).Msg("Rollback: Error updateContext.Runner.Cancel")
				return err
			}
		}

		updateContext.Runner = nil
		updateContext.Resuming = false
	} else {
		log.Info().Msg("Rollback: No installation to cancel")
	}

	updateContext.Reason = "Rolling back to " + updateContext.CurrentTarget.Path
	updateContext.Target = updateContext.CurrentTarget
	updateRunner, err := update.NewUpdate(updateContext.ComposeConfig, updateContext.Target.Path+"|"+updateContext.CorrelationId)
	if err != nil {
		log.Err(err).Msg("Rollback: Error calling update.NewUpdate")
		return err
	}

	err = FillAndCheckAppsList(updateContext)
	if err != nil {
		log.Err(err).Msg("Rollback: Error calling FillAndCheckAppsList")
		return err
	}

	if updateContext.Target == nil {
		// Target is already running
		log.Info().Msgf("Rollback: Target is already running %v", updateContext.Target)
		return nil
	}

	if len(updateContext.RequiredApps) > 0 {
		err = updateRunner.Init(updateContext.Context, updateContext.RequiredApps)
		if err != nil {
			log.Err(err).Msg("rollback init error")
			return err
		}
	}

	updateStatus := updateRunner.Status()
	// Must be in fetched state
	if updateStatus.State != update.StateFetched && updateStatus.State != update.StateInstalled {
		log.Info().Msgf("rollback wrong state error %v", updateStatus.State)
		return fmt.Errorf("rollback update was not fetched %s", updateStatus.State)
	}

	log.Info().Msgf("Proceeding with rollback. Current update runner state is %v", updateStatus.State)

	updateContext.Runner = updateRunner
	err = InstallTarget(updateContext)
	if err != nil {
		log.Err(err).Msg("rollback error installing target")
		return err
	}
	_, err = StartTarget(updateContext)
	if err != nil {
		log.Err(err).Msg("rollback error starting target")
		return err
	}

	log.Err(err).Msg("rollback done")
	return nil
}

func IsTargetRunning(updateContext *UpdateContext) (bool, error) {
	log.Debug().Msgf("Checking target %v", updateContext.Target.Path)
	if updateContext.Target.Path != updateContext.CurrentTarget.Path {
		log.Debug().Msgf("Running target name (%s) is different than candidate target name (%s)", updateContext.CurrentTarget.Path, updateContext.Target.Path)
		return false, nil
	}

	// updateStatus, err := update.GetLastSuccessfulUpdate(updateContext.ComposeConfig)
	// if err != nil {
	// 	log.Info().Msg("error gettingChecking target last update", err)
	// 	return false, err
	// }

	if len(updateContext.RequiredApps) == 0 {
		log.Debug().Msg("No required apps to check")
		return true, nil
	}

	if isSublist(updateContext.InstalledApps, updateContext.RequiredApps) {
		log.Debug().Msg("Installed applications match selected target apps")
		status, err := compose.CheckAppsStatus(updateContext.Context, updateContext.ComposeConfig, updateContext.RequiredApps)
		if err != nil {
			log.Err(err).Msg("Error checking apps status")
			return false, err
		}

		if status.AreRunning() {
			log.Info().Msg("Required applications are are running")
			return true, nil
		} else {
			log.Info().Msgf("Required applications are not running: %v", status.NotRunningApps)
			return false, nil
		}
	} else {
		log.Debug().Msg("Installed applications list do not contain all target apps")
		return false, nil
	}
}

func isSublist[S ~[]E, E comparable](mainList, sublist S) bool {
	if len(sublist) > len(mainList) {
		return false
	}
	for _, subElem := range sublist {
		if !slices.Contains(mainList, subElem) {
			return false
		}
	}
	return true
}

func appsListMatch(appsList1 []string, appsList2 []string) bool {
	if len(appsList1) != len(appsList2) {
		return false
	}

	for _, app1 := range appsList1 {
		found := false
		for _, app2 := range appsList2 {
			if app1 == app2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// func progress(status *update.InitProgress) {
// 	if status.State == update.UpdateInitStateLoadingTree {
// 	} else {
// 	}
// }

func getInstalledApps(updateContext *UpdateContext) ([]string, error) {
	ret := []string{}
	apps, err := compose.ListApps(updateContext.Context, updateContext.ComposeConfig)
	if err != nil {
		log.Err(err).Msg("Error listing apps")
		return nil, fmt.Errorf("error listing apps: %w", err)
	}
	for _, app := range apps {
		if app.Name() != "" {
			ret = append(ret, app.Ref().Spec.Locator+"@"+app.Ref().Digest.String())
		}
	}
	return ret, nil
}
