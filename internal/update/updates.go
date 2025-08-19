// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause
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
	_ "modernc.org/sqlite"
)

type (
	UpdateOptions struct {
		SrcDir    string
		EnableTuf bool
		TargetId  string

		DoCheck   bool
		DoFetch   bool
		DoInstall bool
		DoStart   bool
	}
)

func GetPendingUpdate(updateContext *UpdateContext) error {
	updateRunner, err := update.GetCurrentUpdate(updateContext.ComposeConfig)
	if errors.Is(err, update.ErrUpdateNotFound) {
		log.Debug().Msg("No pending update found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error getting current update: %w", err)
	}

	updateStatus := updateRunner.Status()
	log.Debug().Msgf("Pending update: %v", updateStatus)

	clientRef := updateStatus.ClientRef
	clientRefSplit := strings.Split(clientRef, "|")
	if updateStatus.State == update.StateInitializing || updateStatus.State == update.StateCreated {
		log.Info().Msgf("Canceling current update that was not initialized")
		err = updateRunner.Cancel(updateContext.Context)
		if err != nil {
			log.Warn().Msgf("Error cancelling update: %v", err)
		}
	} else if (clientRefSplit == nil) || (len(clientRefSplit) != 2) {
		log.Warn().Msgf("Invalid clientRef: %s", clientRef)
		err = updateRunner.Cancel(updateContext.Context)
		if err != nil {
			log.Warn().Msgf("Error cancelling update: %v", err)
		}
	} else {
		updateContext.PendingRunner = updateRunner
		updateContext.PendingTargetName = clientRefSplit[0]
		updateContext.PendingCorrelationId = clientRefSplit[1]
		updateContext.PendingApps = updateStatus.URIs
		log.Debug().Msgf("Pending target name: %s, correlation ID: %s, state: %s, pendingApps: %v", updateContext.PendingTargetName, updateContext.PendingCorrelationId, updateStatus.State, updateContext.PendingApps)
	}

	return nil
}

func InitUpdate(updateContext *UpdateContext) error {
	if updateContext.PendingRunner != nil {
		updateContext.Resuming = true
		updateContext.Runner = updateContext.PendingRunner
		updateContext.CorrelationId = updateContext.PendingCorrelationId
	} else {
		log.Info().Msgf("Initializing update for target %s", updateContext.Target.Path)
		version, err := GetVersion(updateContext.Target)
		if err != nil {
			return fmt.Errorf("error getting version: %w", err)
		}
		updateContext.CorrelationId = fmt.Sprintf("%d-%d", version, time.Now().Unix())

		updateRunner, err := update.NewUpdate(updateContext.ComposeConfig, updateContext.Target.Path+"|"+updateContext.CorrelationId)
		if err != nil {
			return err
		}

		initOptions := []update.InitOption{
			update.WithInitProgress(update.GetInitProgressPrinter()),
			update.WithInitAllowEmptyAppList(true),
			update.WithInitCheckStatus(false)}

		err = updateRunner.Init(updateContext.Context, updateContext.RequiredApps, initOptions...)
		if err != nil {
			return err
		}
		us := updateRunner.Status()
		if len(us.URIs) > 0 {
			fmt.Printf("Diff summary:\t\t\t\t  %d blobs (%s) to fetch\n", len(us.Blobs), compose.FormatBytesInt64(us.TotalBlobsBytes))
		}
		log.Debug().Msgf("Initialized new update. Status: %v, CorrelationId: %s", us.State, updateContext.CorrelationId)
		updateContext.Runner = updateRunner
	}
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

	fetchOptions := []compose.FetchOption{
		compose.WithFetchProgress(update.GetFetchProgressPrinter()),
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

func InstallTarget(updateContext *UpdateContext) error {
	updateStatus := updateContext.Runner.Status()
	if updateStatus.State != update.StateFetched && updateStatus.State != update.StateInstalling {
		log.Debug().Msgf("update was already installed. Update state: %s", updateStatus.State)
		if updateContext.Resuming {
			return nil
		}
	}

	targets.RegisterInstallationStarted(updateContext.DbFilePath, updateContext.Target, updateContext.CorrelationId)
	err := GenAndSaveEvent(updateContext, events.InstallationStarted, "", nil)
	if err != nil {
		log.Err(err).Msg("error on GenAndSaveEvent")
	}

	installOptions := []compose.InstallOption{
		compose.WithInstallProgress(update.GetInstallProgressPrinter())}

	if len(updateContext.AppsToUninstall) > 0 {
		log.Info().Msgf("Stopping apps not included in target %v", updateContext.Target.Path)
		log.Debug().Msgf("Apps being stopped: %v", updateContext.AppsToUninstall)
		err = compose.StopApps(updateContext.Context, updateContext.ComposeConfig, updateContext.AppsToUninstall)
		if err != nil {
			log.Err(err).Msg("error stopping apps before installing target")
		}
	}

	log.Info().Msgf("Installing target %v", updateContext.Target.Path)
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
	log.Info().Msgf("Starting target %v", updateContext.Target.Path)

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
		errEvt := GenAndSaveEvent(updateContext, events.InstallationCompleted, err.Error(), targets.BoolPointer(false))
		if errEvt != nil {
			log.Err(errEvt).Msg("error on GenAndSaveEvent")
		}
		targets.RegisterInstallationFailed(updateContext.DbFilePath, updateContext.Target, updateContext.CorrelationId)
		// rollback(updateContext)
		return true, fmt.Errorf("error starting target: %w", err)
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

	log.Info().Msgf("Target %v has been started", updateContext.Target.Path)
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

	initOptions := []update.InitOption{update.WithInitAllowEmptyAppList(true), update.WithInitCheckStatus(false)}
	err = updateRunner.Init(updateContext.Context, updateContext.RequiredApps, initOptions...)
	if err != nil {
		log.Err(err).Msg("rollback init error")
		return err
	}

	updateStatus := updateRunner.Status()
	if updateStatus.State != update.StateInitialized {
		log.Info().Msgf("rollback unexpected state error %v", updateStatus.State)
		return fmt.Errorf("rollback update was %s, expected initialized", updateStatus.State)
	}

	// Call fetch just to move the update to the next state. No actual data should be fetched, and no events should be generated
	err = updateRunner.Fetch(updateContext.Context)
	if err != nil {
		log.Err(err).Msg("rollback fetch error")
		return fmt.Errorf("rollback update fetch error: %w", err)
	}

	updateContext.Runner = updateRunner
	log.Info().Msgf("Installing rollback target %v", updateContext.Target.Path)
	err = InstallTarget(updateContext)
	if err != nil {
		log.Err(err).Msg("rollback error installing target")
		return err
	}

	log.Info().Msgf("Starting rollback target %v", updateContext.Target.Path)
	_, err = StartTarget(updateContext)
	if err != nil {
		log.Err(err).Msgf("rollback error starting target %v", updateContext.Target.Path)
		return err
	}
	log.Info().Msgf("Rollback to target %v completed successfully", updateContext.Target.Path)
	return nil
}

func IsTargetRunning(updateContext *UpdateContext) (bool, error) {
	log.Debug().Msgf("Checking target %v", updateContext.Target.Path)
	if updateContext.Target.Path != updateContext.CurrentTarget.Path {
		log.Debug().Msgf("Running target name (%s) is different than candidate target name (%s)", updateContext.CurrentTarget.Path, updateContext.Target.Path)
		return false, nil
	}

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
