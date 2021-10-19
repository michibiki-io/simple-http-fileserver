#!/bin/bash
set -euo pipefail

map_uidgid() {
    USERMAP_ORIG_UID=$(id -u go)
    USERMAP_ORIG_GID=$(id -g go)
    USERMAP_GID=${USERMAP_GID:-$USERMAP_ORIG_GID}
    USERMAP_UID=${USERMAP_UID:-$USERMAP_ORIG_UID}
    if [ "${USERMAP_UID}" != "${USERMAP_ORIG_UID}" ] || [ "${USERMAP_GID}" != "${USERMAP_ORIG_GID}" ]; then
        echo "Starting with UID : $USERMAP_UID, GID: $USERMAP_GID"
        usermod -u $USERMAP_UID -o go
        groupmod -g $USERMAP_GID go
        chown go:go /opt/go
    fi
}

if [ "$(id -u)" = '0' ]; then
    map_uidgid
    exec gosu go "$@"
else
    exec "$@"
fi
