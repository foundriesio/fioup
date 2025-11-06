# Getting Started With Fioup

## Installing Fioup

### Official Debian Package (Recommended)

1. Update the `apt` package index and install packages needed to use the fioup `apt` repository:

   ```
   sudo apt update
   sudo apt install -y apt-transport-https ca-certificates curl gnupg
   ```

1. Download the public signing key for the package repositories:

   ```
   curl -L https://fioup.foundries.io/pkg/deb/dists/stable/Release.gpg | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/fioup-stable.gpg
   ```

1. Add the appropriate `apt` repository.

   ```
   echo 'deb [signed-by=/etc/apt/trusted.gpg.d/fioup-stable.gpg] https://fioup.foundries.io/pkg/deb stable main' | sudo tee /etc/apt/sources.list.d/fioup.list
   ```

1. Install fioup

   ```
   sudo apt update
   sudo apt install fioup
   ```

A systemd service, `fioup`, is created in a disabled state. This service
can optionally be enabled after [registering](./register-device.md) a device.

### GitHub Releases (Advanced)

> [!NOTE]
> These steps are described for a non-root user.

Statically compiled binaries are created with each GitHub [release](https://github.com/foundriesio/fioup/releases).
The binary must be placed in the user's `PATH`. If the directory _is not_
writable by the user, then they must manually create a symlink:

```
 # Assuming /usr/local/bin/fioup
 sudo ln -s /usr/local/bin/fioup /usr/local/bin/docker-credential-fioup
```
