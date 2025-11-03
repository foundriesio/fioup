# Remote configuration

FoundriesFactory includes a feature for managing device configuration
remotely known as "[fioconfig](https://docs.foundries.io/latest/reference-manual/ota/configuring.html)".
Fioup has built-in support for this feature.

## Prerequisites

The device must be [registered](./register-device.md) with a Factory as
**root** to use remote configuration functions.

## Manual usage

Fioup includes a few ways to work with configuration data.

### config-check

`fioup config-check` makes a request to the [Device Gateway](https://docs.foundries.io/latest/reference-manual/ota/ota-architecture.html)
to check for new configuration data. If configuration has changed, it will
download and extract the data.

### config-extract

Devices often need access to configuration data in the boot process before
networking is available and the daemon might run. `fioup config-extract` will
decrypt and extract the current copy of configuration data, `/var/sota/config.encrypted`,
to `/run/secrets` if its availble.

### daemon mode

By default, `fioup daemon` checks with the Device Gateway for new configuration
data during each update interval. This can be disabled with `fioup daemon --fioconfig=0`.

## Example

The most common use of fioconfig is configuring the list of apps a device
should run. This can be done in the web UI or with fioctl from your host
computer with:
```
 $ fioctl devices config updates <device> --apps comma,separated,list
```

## Further reading

 * Fioconfig [API, storage, and encryption details](https://docs.foundries.io/latest/reference-manual/ota/fioconfig.html)
 * [Example](https://docs.foundries.io/95/tutorials/configuring-and-sharing-volumes/dynamic-configuration-file.html) configuring a container
