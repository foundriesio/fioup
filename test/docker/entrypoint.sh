#!/bin/sh -e

if [ -z $DEV_USER ] || [ -z $DEV_GROUP ]; then
    echo "DEV_USER and DEV_GROUP environment variables must be set."
    exit 1
fi

# Create a group with the specified GID if it doesn't already exist
if ! getent group $DEV_GROUP >/dev/null; then
    groupadd -g $DEV_GROUP devgrp
fi

# Create a user with the specified UID and GID if it doesn't already exist
if ! getent passwd $DEV_USER >/dev/null; then
    useradd -u $DEV_USER -g $DEV_GROUP -m dev
fi

# Change ownership of the home directory to the appuser
chown -R dev:devgrp /home/dev

chown dev:devgrp /var/run/docker/docker.sock

chown -R dev:devgrp /var/sota
chown -R dev:devgrp /usr/lib/sota/conf.d
chown -R dev:devgrp /etc/sota/conf.d
chown -R dev:devgrp /var/lib/docker

ln -sfn ${PWD}/bin/fioup /usr/local/bin/fioup
ln -sfn ${PWD}/bin/fioup /usr/local/bin/docker-credential-fioup
chmod +x ${PWD}/debian/fioconfig-handlers/aktualizr-toml-update
ln -sfn ${PWD}/debian/fioconfig-handlers/aktualizr-toml-update /usr/share/fioconfig/handlers/aktualizr-toml-update

# Run the command as the created user
exec gosu $DEV_USER:$DEV_GROUP "$@"
