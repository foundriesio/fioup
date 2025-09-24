# Daemon mode

Daemon mode allows a device to run Fioup as a SystemD service periodically
checking in with the Foundries.io backend to apply updates as they are built
in a Factory. It is disabled by default.

## Prerequisites

Before enabling daemon mode, the device must be [registered](./register-device.md)
with a Factory as **root**.

## Enabling

The SystemD service can be enabled and started with:
```
 $ sudo systemctl enable fioup
 $ sudo systemctl start fioup
```

The daemon can also be run by hand with `sudo fioup daemon`.

## Advanced options

The default polling interval for updates is every 300 seconds. This can be
configured in `/var/sota/sota.toml` by adding:
```
[uptane]
polling_seconds = 60
```
