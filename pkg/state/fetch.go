// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/pkg/errors"
)

type (
	Fetch struct {
		ProgressHandler compose.FetchProgressFunc
	}

	InsufficientStorageError struct {
		UsageInfo *StorageStat
	}
)

var (
	ErrFetchFailed  = errors.New("download failed")
	ErrFetchNoSpace = errors.New("download failed, not enough storage space")
)

func (s *Fetch) Name() ActionName { return "Fetching" }
func (s *Fetch) Execute(ctx context.Context, updateCtx *UpdateContext) error {
	var err error
	updateState := updateCtx.UpdateRunner.Status().State
	switch updateState {
	case update.StateCreated, update.StateInitializing:
		return fmt.Errorf("%w:  update not initialized, cannot fetch", ErrInvalidActionForState)
	case update.StateInitialized, update.StateFetching:
		// Update storage usage info before fetch to reflect current usage
		if errUsage := updateCtx.getAndSetStorageUsageInfo(); errUsage != nil {
			slog.Debug("failed to get storage usage info after fetch", "error", errUsage)
		} else {
			// Check for free space only if we could get current storage usage info
			err = updateCtx.checkFreeSpace()
			if err != nil {
				err = fmt.Errorf("%w: %w", ErrFetchNoSpace, err)
			}
		}
		// Send download started event regardless if there is enough space or not to mark the start of download attempt
		updateCtx.SendEvent(events.DownloadStarted)
		if err == nil {
			err = updateCtx.UpdateRunner.Fetch(ctx, compose.WithFetchProgress(s.ProgressHandler))
			if err != nil {
				err = fmt.Errorf("%w: %w", ErrFetchFailed, err)
			}
			// Update storage usage info after fetch to reflect actual usage
			if errInfo := updateCtx.getAndSetStorageUsageInfo(); errInfo != nil {
				slog.Debug("failed to get storage usage info after fetch", "error", errInfo)
			}
		}
		updateCtx.SendEvent(events.DownloadCompleted, err)
	default:
		updateCtx.AlreadyFetched = true
	}
	return err
}

func (u *UpdateContext) checkFreeSpace() error {
	updateStatus := u.UpdateRunner.Status()
	var requiredBytes int64
	var requiredBytesTotal uint64

	for _, blob := range updateStatus.Blobs {
		requiredBytes += blob.StoreSize - compose.AlignToBlockSize(blob.BytesFetched, u.Config.ComposeConfig().BlockSize)
		// The runtime size is a size of an uncompressed blob loaded and stored in the docker engine store, hence
		// assumption is that a given blob is not loaded at all even if it is partially fetched.
		requiredBytes += blob.RuntimeSize
	}
	requiredBytesTotal = uint64(requiredBytes)
	u.StorageUsage.Required = &requiredBytesTotal
	if u.StorageUsage.Required != nil && *u.StorageUsage.Required > u.StorageUsage.Available {
		return &InsufficientStorageError{UsageInfo: u.StorageUsage}
	}
	return nil
}

func (e *InsufficientStorageError) Error() string {
	var required uint64
	if e.UsageInfo.Required != nil {
		required = *e.UsageInfo.Required
	}
	return fmt.Sprintf("not enough space for update: required %s, available %s; storage: size %s, free %s, reserved %s",
		compose.FormatBytesUint64(required), compose.FormatBytesUint64(e.UsageInfo.Available),
		compose.FormatBytesUint64(e.UsageInfo.Size), compose.FormatBytesUint64(e.UsageInfo.Free),
		compose.FormatBytesUint64(e.UsageInfo.Reserved))

}
