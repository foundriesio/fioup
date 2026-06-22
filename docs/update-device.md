# Updating a Device

## Prerequisites

This document assumes an application has been built in the Factory. If
this is not the case, please follow the documentation for
[building and deploying application](https://docs.foundries.io/latest/getting-started/building-deploying-app/index.html).

## Applying the Update

You can verify if an update is available by running `sudo fioup check`. If an
update is available, it may be applied by running:

```
 sudo fioup update
```

Optionally, you can check the update size before applying the update by running:

```
sudo fioup diff
```

## Advanced

An update can be applied in more granular steps with:

```
 sudo fioup check
 sudo fioup fetch
 sudo fioup install
 sudo fioup start
```

> [!IMPORTANT]
> Once you've started an update sequence, you must `fioup cancel` to start a new sequence.

The update status can be checked at any time with `sudo fioup status`.

### Configure Image Pruning Mode

By default, once updated apps have been started, `fioup` prunes only unused container images associated with apps
managed by `fioup`.

In this context, *image pruning* means removing image references left behind after an app has been removed or updated.
If the removed reference is the last one pointing to that image, Docker deletes the image itself.

`fioup` can also be configured to prune **all** unused images, including images not related to apps managed by `fioup`.

To enable this mode, set the `pacman.prune_unused_images` option in the `fioup` configuration file (by
default `/var/sota/sota.toml`) to `"1"`:

```toml
[pacman]
prune_unused_images = "1"
```

### Configure Storage Usage

`fioup` checks that enough storage is available before fetching and installing
an update, and refuses to proceed otherwise. Two options under the `[pacman]`
section of the `fioup` configuration file (by default `/var/sota/sota.toml`)
control how much of the device's storage may be consumed by apps.

#### `storage_watermark`

A percentage of the total storage that apps may use. The value must be an
integer between `20` and `99`; when unset, the default is `95`.

```toml
[pacman]
storage_watermark = "80"
```

With the example above, an update is allowed only while the total storage
usage including the apps stays below 80% of the underlying filesystem.

#### `reserved_storage`

An absolute amount of free space to keep reserved for non-app usage, expressed
as a byte size with either a binary (`KiB`, `MiB`, `GiB`) or decimal (`KB`,
`MB`, `GB`) suffix. When set, apps may use all available storage except the
reserved amount.

```toml
[pacman]
reserved_storage = "2GiB"
```

`reserved_storage` takes precedence over `storage_watermark`: when both are
set, the percentage watermark is ignored and a warning is logged. Use
`reserved_storage` on devices with large data partitions where a fixed
absolute reserve is more meaningful than a percentage.
