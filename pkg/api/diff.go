package api

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/platforms"
	"github.com/foundriesio/composeapp/pkg/compose"
	"github.com/foundriesio/composeapp/pkg/update"
	"github.com/foundriesio/fioup/pkg/client"
	"github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/target"
)

type (
	DiffOptions struct {
		EnableTUF bool
	}
	DiffOption func(*DiffOptions)

	DiffReport struct {
		FromTarget *target.Target `json:"from_target"`
		ToTarget   *target.Target `json:"to_target"`
		WireSize   int64          `json:"wire_size"`
		DiskSize   int64          `json:"disk_size"`
		// Blobs maps app base URI to list of blobs info, grouping by app
		Blobs map[string][]compose.BlobInfo `json:"blobs"`
	}
)

func WithTUFEnabled(enabled bool) DiffOption {
	return func(opts *DiffOptions) {
		opts.EnableTUF = enabled
	}
}

func Diff(ctx context.Context, cfg *config.Config, fromVersion, toVersion int, options ...DiffOption) (*DiffReport, error) {
	opts := DiffOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	gwClient, err := client.NewGatewayClient(cfg, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway client: %w", err)
	}

	var targetRepo target.Repo
	if opts.EnableTUF {
		targetRepo, err = target.NewTufRepo(cfg, gwClient, cfg.GetHardwareID())
	} else {
		targetRepo, err = target.NewPlainRepo(gwClient, cfg.GetTargetsFilepath(), cfg.GetHardwareID())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create target repo: %w", err)
	}

	var targets target.Targets
	if targets, err = targetRepo.LoadTargets(false); err != nil {
		return nil, fmt.Errorf("failed to load targets: %w", err)
	}

	shortlistApps := fromVersion == -1
	var fromTarget target.Target
	if fromVersion == -1 {
		if lastUpdate, err := update.GetLastSuccessfulUpdate(cfg.ComposeConfig()); err == nil {
			fromTarget = targets.GetTargetByID(lastUpdate.ClientRef)
		} else {
			return nil, fmt.Errorf("failed to get current target: %w", err)
		}
	} else {
		fromTarget = targets.GetTargetByVersion(fromVersion)
	}
	var toTarget target.Target
	if toVersion == -1 {
		toTarget = targets.GetLatestTarget()
	} else {
		toTarget = targets.GetTargetByVersion(toVersion)
	}

	if toTarget.IsUnknown() {
		return nil, fmt.Errorf("failed to find target for version %d", toVersion)
	}
	if fromTarget.IsUnknown() && fromVersion != -1 {
		return nil, fmt.Errorf("failed to find target for version %d", fromVersion)
	}
	if shortlistApps {
		fromTarget.ShortlistApps(cfg.GetEnabledApps())
		toTarget.ShortlistApps(cfg.GetEnabledApps())
	}

	var toBlobs map[string]compose.BlobInfo
	toBlobs, err = getTargetAppBlobs(ctx, cfg, toTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to get app blobs for target version %d: %w", toVersion, err)
	}

	var fromBlobs map[string]compose.BlobInfo
	if fromVersion == -1 {
		fromBlobs, err = getCurrentAppBlobs(ctx, cfg, fromTarget.AppURIs())
	} else {
		fromBlobs, err = getTargetAppBlobs(ctx, cfg, fromTarget)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app blobs for target version %d: %w", fromVersion, err)
	}

	diff := DiffReport{
		FromTarget: &fromTarget,
		ToTarget:   &toTarget,
		Blobs:      make(map[string][]compose.BlobInfo),
	}
	for blobHash, blobInfo := range toBlobs {
		if _, found := fromBlobs[blobHash]; !found {
			diff.WireSize += blobInfo.Descriptor.Size
			diff.DiskSize += blobInfo.StoreSize + blobInfo.RuntimeSize
			ref, _ := compose.ParseImageRef(blobInfo.Ref())
			diff.Blobs[ref.Locator] = append(diff.Blobs[ref.Locator], blobInfo)
		}
	}

	return &diff, nil
}

func getCurrentAppBlobs(ctx context.Context, cfg *config.Config, appURIs []string) (map[string]compose.BlobInfo, error) {
	status, err := compose.CheckAppsStatus(ctx, cfg.ComposeConfig(), appURIs, compose.WithQuickCheckFetch(true),
		compose.WithCheckInstallation(false), compose.WithCheckRunning(false))
	if err != nil {
		return nil, fmt.Errorf("failed to check current target's apps status: %w", err)
	}
	return getAppBlobs(status.Apps, cfg.ComposeConfig().Platform.Architecture, cfg.ComposeConfig().BlockSize)
}

func getTargetAppBlobs(ctx context.Context, cfg *config.Config, target target.Target) (map[string]compose.BlobInfo, error) {
	var apps []compose.App
	srcBlobProvider := compose.NewRemoteBlobProviderFromConfig(cfg.ComposeConfig())
	for _, appURI := range target.AppURIs() {
		if app, err := cfg.ComposeConfig().AppLoader.LoadAppTree(ctx, srcBlobProvider, platforms.OnlyStrict(cfg.ComposeConfig().Platform), appURI); err != nil {
			return nil, fmt.Errorf("failed to load app %s: %w", appURI, err)
		} else {
			apps = append(apps, app)
		}
	}
	return getAppBlobs(apps, cfg.ComposeConfig().Platform.Architecture, cfg.ComposeConfig().BlockSize)
}

func getAppBlobs(apps []compose.App, arch string, blockSize int64) (map[string]compose.BlobInfo, error) {
	blobs := make(map[string]compose.BlobInfo)
	for _, app := range apps {
		err := app.Tree().Walk(func(node *compose.TreeNode, depth int) error {
			blobStoreSize := compose.AlignToBlockSize(node.Descriptor.Size, blockSize)
			blobRuntimeSize := app.GetBlobRuntimeSize(node.Descriptor, arch, blockSize)

			blobs[node.Descriptor.Digest.String()] = compose.BlobInfo{
				Descriptor:  node.Descriptor,
				StoreSize:   blobStoreSize,
				RuntimeSize: blobRuntimeSize,
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return blobs, nil
}
