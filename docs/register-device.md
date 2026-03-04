# Registering a Device

This section assumes you have created a Factory. Please follow
[sign up process](https://docs.foundries.io/latest/getting-started/signup/index.html)
if not.

The registration command does several things:

* Creates a mTLS signing request for the FoundriesFactory™ Platform backend.
* Creates a device entry in the backend.
* Stores configuration and connection material under `/var/sota`.
* Ensures the Docker credential helper, `docker-credential-fioup`, is
  present. If not, it will try to symlink `fioup` -> `docker-credential-fioup`.
* Configure `$HOME/.docker/config.json` to use this helper for
  authenticating with `hub.foundries.io`.

To register a device run the following command:

```
 sudo fioup register --factory <FACTORY_NAME> --name <NAME_FOR_DEVICE>
```

## Example

```
 sudo fioup register --factory example-factory --name device-1
```

```
 Visit the link below in your browser to authorize this new device. This link will expire in 15 minutes.
  Device UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  User code: xxxx-xxxx
  Browser URL: https://app.fioed.io/activate/

 Device is now registered. -
```

## Verifying Registration

The `check` subcommand can be used to verify connectivity with the
FoundriesFactory backend:

```
 $ sudo fioup check
```

## Changing Apps That Run

The default behavior of `fioup` is to run all apps defined in a Target. This
can be overridden by setting the `pacman.compose_apps` field in
`/var/sota/sota.toml` to a comma separated list of applications. Examples
include:

```
[pacman]
# Run two apps, foo and bar
compose_apps = "foo,bar"

# Run no apps:
compose_apps = " "
```

Fioup parses and merges configuration options using the follow logic:

* `/usr/lib/sota/conf.d/*.toml`
* `/var/sota/sota.toml`
* `/etc/sota/conf.d/*.toml`

This provides the user a framework for overriding configuration options.
