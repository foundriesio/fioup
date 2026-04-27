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
