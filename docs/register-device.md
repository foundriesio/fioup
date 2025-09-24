# Registering a device

This section assumes you have created a FoundriesFactory. Please follow
[sign up process](https://docs.foundries.io/latest/getting-started/signup/index.html)
if not.

The registration command does several things:
 * Creates a mTLS signing request for the Foundries backend.
 * Creates a device entry in the Foundries backend.
 * Stores configuration and connection material under `/var/sota`.
 * Ensures the Docker credential helper, `docker-credential-fioup`, is
   present. If not, it will try to symlink `fioup` -> `docker-credential-fioup`.
 * Configure `$HOME/.docker/config.json` to use this helper for
   authenticating with `hub.foundries.io`.


## As root (recommened)
```
 $ sudo fioup register --factory <FACTORY_NAME> --name <NAME_FOR_DEVICE>
```

## As non-root user (advanced)
```
 $ sudo chown -r $USER /var/sota
 $ fioup register --factory <FACTORY_NAME> --name <NAME_FOR_DEVICE>
```

**NOTE** If the credential helper isn't installed (i.e. fioup was not
installed with as a Debian package), then the user must have a writable
directory in their `PATH` for fioup to setup the credential helper.
Altentatively, you can create a symlink with something like:
```
 $ sudo ln -s <path to fioup> /usr/local/bin/docker-credential-fioup
```

## Example
```
 $ sudo fioup register --factory example-factory --name device-1
 Visit the link below in your browser to authorize this new device. This link will expire in 15 minutes.
  Device UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  User code: xxxx-xxxx
  Browser URL: https://app.fioed.io/activate/

 Device is now registered. -
```

## Verifying registration
The `check` subcommand can be used to verifiy connectivity with the
Foundries.io backend:
```
 $ sudo fioup check
```
