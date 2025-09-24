// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package update

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/internal/targets"
	"github.com/foundriesio/fioup/pkg/fioup/target"
	_ "modernc.org/sqlite"
)

type (
	UpdateOptions struct {
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
		slog.Debug("No pending update found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error getting current update: %w", err)
	}

	updateStatus := updateRunner.Status()
	slog.Debug(fmt.Sprintf("Pending update: %v", updateStatus))

	switch updateStatus.State {
	case update.StateStarted:
		slog.Debug("Completing current update that was started")
		err = updateRunner.Complete(updateContext.Context, update.CompleteWithPruning())
		if err != nil {
			slog.Warn(fmt.Sprintf("Error completing update: %v", err))
		}
	case update.StateInitializing, update.StateCreated:
		slog.Info("Canceling current update that was not initialized")
		err = updateRunner.Cancel(updateContext.Context)
		if err != nil {
			slog.Warn(fmt.Sprintf("Error cancelling update: %v", err))
		}
	default:
		updateContext.PendingRunner = updateRunner
		updateContext.PendingTargetName = updateStatus.ClientRef
		updateContext.PendingApps = updateStatus.URIs
		slog.Debug(fmt.Sprintf("Pending target name: %s, correlation ID: %s, state: %s, pendingApps: %v", updateContext.PendingTargetName, updateRunner.Status().ID, updateStatus.State, updateContext.PendingApps))
	}

	return nil
}

func InitUpdate(updateContext *UpdateContext) error {
	if updateContext.PendingRunner != nil {
		updateContext.Resuming = true
		updateContext.Runner = updateContext.PendingRunner
	} else {
		slog.Info(fmt.Sprintf("Initializing update for target %s", updateContext.Target.ID))
		updateRunner, err := update.NewUpdate(updateContext.ComposeConfig, updateContext.Target.ID)
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
		slog.Debug(fmt.Sprintf("Initialized new update. Status: %v, CorrelationId: %s", us.State, us.ID))
		updateContext.Runner = updateRunner
	}
	return nil
}

func PullTarget(updateContext *UpdateContext) error {
	slog.Info(fmt.Sprintf("Pulling target %v", updateContext.Target.ID))

	var updateStatus update.Update
	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateInitialized && updateStatus.State != update.StateFetching {
		slog.Info(fmt.Sprintf("update has already been fetched. Update state: %s", updateStatus.State))
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
		errEvt := GenAndSaveEvent(updateContext, events.DownloadCompleted, err.Error(), targets.BoolPointer(false))
		if errEvt != nil {
			slog.Error("error on GenAndSaveEvent", "error", errEvt)
		}
		return fmt.Errorf("error pulling target: %w", err)
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateFetched {
		slog.Info("update not fetched")
	}
	if updateStatus.Progress != 100 {
		slog.Info(fmt.Sprintf("update is not fetched for 100%%: %d", updateStatus.Progress))
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
		slog.Debug(fmt.Sprintf("update was already installed. Update state: %s", updateStatus.State))
		if updateContext.Resuming {
			return nil
		}
	}

	err := targets.RegisterInstallationStarted(updateContext.DbFilePath, &updateContext.Target, updateStatus.ID)
	if err != nil {
		slog.Error("error registering installation started", "error", err)
	}

	err = GenAndSaveEvent(updateContext, events.InstallationStarted, "", nil)
	if err != nil {
		slog.Error("error on GenAndSaveEvent", "error", err)
	}

	installOptions := []compose.InstallOption{
		compose.WithInstallProgress(update.GetInstallProgressPrinter())}

	if len(updateContext.AppsToUninstall) > 0 {
		slog.Info(fmt.Sprintf("Stopping apps not included in target %v", updateContext.Target.ID))
		slog.Debug(fmt.Sprintf("Apps being stopped: %v", updateContext.AppsToUninstall))
		err = compose.StopApps(updateContext.Context, updateContext.ComposeConfig, updateContext.AppsToUninstall)
		if err != nil {
			slog.Error("error stopping apps before installing target", "error", err)
		}
	}

	slog.Info(fmt.Sprintf("Installing target %v", updateContext.Target.ID))
	err = updateContext.Runner.Install(updateContext.Context, installOptions...)
	if err != nil {
		if err2 := GenAndSaveEvent(updateContext, events.DownloadCompleted, err.Error(), targets.BoolPointer(false)); err2 != nil {
			err = errors.Join(err, err2)
		}
		return fmt.Errorf("error installing target: %w", err)
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.State != update.StateInstalled {
		slog.Debug("update not installed")
	}
	if updateStatus.Progress != 100 {
		slog.Debug(fmt.Sprintf("update is not installed for 100%%: %d", updateStatus.Progress))
	}

	err = GenAndSaveEvent(updateContext, events.InstallationApplied, "", nil)
	if err != nil {
		slog.Error("error on GenAndSaveEvent", "error", err)
	}
	return nil
}

func StartTarget(updateContext *UpdateContext) (bool, error) {
	slog.Info(fmt.Sprintf("Starting target %v", updateContext.Target.ID))

	var err error
	updateStatus := updateContext.Runner.Status()
	if updateStatus.State != update.StateInstalled && updateStatus.State != update.StateStarting {
		slog.Debug(fmt.Sprintf("Skipping start target operation because state is: %s", updateStatus.State))
		if updateContext.Resuming {
			return false, nil
		}
	}

	err = compose.StopApps(updateContext.Context, updateContext.ComposeConfig, updateContext.AppsToUninstall)
	if err != nil {
		slog.Error("error stopping apps before starting target", "error", err)
	}

	err = updateContext.Runner.Start(updateContext.Context)
	if err != nil {
		slog.Error("error on starting target", "error", err)
		errEvt := GenAndSaveEvent(updateContext, events.InstallationCompleted, err.Error(), targets.BoolPointer(false))
		if errEvt != nil {
			slog.Error("error on GenAndSaveEvent", "error", errEvt)
		}

		errDb := targets.RegisterInstallationFailed(updateContext.DbFilePath, &updateContext.Target, updateStatus.ID)
		if errDb != nil {
			slog.Error("error registering installation failed", "error", errDb)
		}
		// rollback(updateContext)
		return true, fmt.Errorf("error starting target: %w", err)
	}

	if updateContext.Runner.Status().State != update.StateStarted {
		slog.Info("update not started")
	}

	updateStatus = updateContext.Runner.Status()
	if updateStatus.Progress != 100 {
		slog.Debug(fmt.Sprintf("update is not started for 100%%: %d", updateStatus.Progress))
	}

	err = GenAndSaveEvent(updateContext, events.InstallationCompleted, "", targets.BoolPointer(true))
	if err != nil {
		slog.Error("error on GenAndSaveEvent", "error", err)
	}
	err = targets.RegisterInstallationSuceeded(updateContext.DbFilePath, &updateContext.Target, updateStatus.ID)
	if err != nil {
		slog.Error("error registering installation succeeded", "error", err)
	}

	slog.Debug("Completing update with pruning")
	err = updateContext.Runner.Complete(updateContext.Context, update.CompleteWithPruning())
	if err != nil {
		slog.Error("error completing update:", "error", err)
	}

	slog.Info(fmt.Sprintf("Target %v has been started", updateContext.Target.ID))
	return false, nil
}

func rollback(updateContext *UpdateContext) error {
	slog.Info(fmt.Sprintf("Rolling back to target %v", updateContext.CurrentTarget.ID))
	if updateContext.Runner != nil {
		updateStatus := updateContext.Runner.Status()
		if updateStatus.State == update.StateStarted {
			err := updateContext.Runner.Complete(updateContext.Context)
			if err != nil {
				slog.Error("Rollback: Error updateContext.Runner.Complete", "error", err)
			}
		} else if updateStatus.State != update.StateFailed {
			err := updateContext.Runner.Cancel(updateContext.Context)
			if err != nil {
				slog.Error("Rollback: Error updateContext.Runner.Cancel", "error", err)
				return err
			}
		}
		updateContext.Runner = nil
		updateContext.Resuming = false
		updateContext.PendingApps = nil
		updateContext.PendingRunner = nil
		updateContext.PendingTargetName = ""
	} else {
		slog.Info("Rollback: No installation to cancel")
	}

	updateContext.Reason = "Rolling back to " + updateContext.CurrentTarget.ID
	updateContext.Target = updateContext.CurrentTarget

	err := FillAppsList(updateContext)
	if err != nil {
		slog.Error("Rollback: Error calling FillAppsList", "error", err)
	}

	updateRunner, err := update.NewUpdate(updateContext.ComposeConfig, updateContext.Target.ID)
	if err != nil {
		slog.Error("Rollback: Error calling update.NewUpdate", "error", err)
		return err
	}

	err = FillAndCheckAppsList(updateContext)
	if err != nil {
		slog.Error("Rollback: Error calling FillAndCheckAppsList", "error", err)
		return err
	}

	if updateContext.Target.ID == target.UnknownTarget.ID {
		// Target is already running
		slog.Info(fmt.Sprintf("Rollback: Target is already running %v", updateContext.Target))
		return nil
	}

	initOptions := []update.InitOption{update.WithInitAllowEmptyAppList(true), update.WithInitCheckStatus(false)}
	err = updateRunner.Init(updateContext.Context, updateContext.RequiredApps, initOptions...)
	if err != nil {
		slog.Error("rollback init error", "error", err)
		return err
	}

	updateStatus := updateRunner.Status()
	if updateStatus.State != update.StateInitialized {
		slog.Info(fmt.Sprintf("rollback unexpected state error %v", updateStatus.State))
		return fmt.Errorf("rollback update was %s, expected initialized", updateStatus.State)
	}

	// Call fetch just to move the update to the next state. No actual data should be fetched, and no events should be generated
	err = updateRunner.Fetch(updateContext.Context)
	if err != nil {
		slog.Error("rollback fetch error", "error", err)
		return fmt.Errorf("rollback update fetch error: %w", err)
	}

	updateContext.Runner = updateRunner
	slog.Info(fmt.Sprintf("Installing rollback target %v", updateContext.Target.ID))
	err = InstallTarget(updateContext)
	if err != nil {
		slog.Error("rollback error installing target", "error", err)
		return err
	}

	slog.Info(fmt.Sprintf("Starting rollback target %v", updateContext.Target.ID))
	_, err = StartTarget(updateContext)
	if err != nil {
		slog.Error(fmt.Sprintf("rollback error starting target %v", updateContext.Target.ID), "error", err)
		return err
	}
	slog.Info(fmt.Sprintf("Rollback to target %v completed successfully", updateContext.Target.ID))
	return nil
}

func IsTargetRunning(updateContext *UpdateContext) (bool, error) {
	slog.Debug(fmt.Sprintf("Checking target %v", updateContext.Target.ID))
	if updateContext.Target.ID != updateContext.CurrentTarget.ID {
		slog.Debug(fmt.Sprintf("Running target name (%s) is different than candidate target name (%s)", updateContext.CurrentTarget.ID, updateContext.Target.ID))
		return false, nil
	}

	if len(updateContext.RequiredApps) == 0 {
		slog.Debug("No required apps to check")
		return true, nil
	}

	if isSublist(updateContext.InstalledApps, updateContext.RequiredApps) {
		slog.Debug("Installed applications match selected target apps")
		status, err := compose.CheckAppsStatus(updateContext.Context, updateContext.ComposeConfig, updateContext.RequiredApps)
		if err != nil {
			slog.Error("Error checking apps status", "error", err)
			return false, err
		}

		if status.AreRunning() {
			slog.Info("Required applications are are running")
			return true, nil
		} else {
			slog.Info(fmt.Sprintf("Required applications are not running: %v", status.NotRunningApps))
			return false, nil
		}
	} else {
		slog.Debug("Installed applications list do not contain all target apps")
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

func getInstalledApps(updateContext *UpdateContext) ([]string, []string, error) {
	retApps := []string{}
	retAppsNames := []string{}
	apps, err := compose.ListApps(updateContext.Context, updateContext.ComposeConfig)
	if err != nil {
		slog.Error("Error listing apps", "error", err)
		return nil, nil, fmt.Errorf("error listing apps: %w", err)
	}
	for _, app := range apps {
		if app.Name() != "" {
			retApps = append(retApps, app.Ref().Spec.Locator+"@"+app.Ref().Digest.String())
			retAppsNames = append(retAppsNames, app.Name())
		}
	}
	return retApps, retAppsNames, nil
}
