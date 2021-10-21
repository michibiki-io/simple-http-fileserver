#!/bin/bash
set -euo pipefail

map_uidgid() {
    USERMAP_ORIG_UID=$(id -u go)
    USERMAP_ORIG_GID=$(id -g go)
    USERMAP_ORIG_GIDS=$(id -G go)
    USERMAP_UID=${USERMAP_UID:-$USERMAP_ORIG_UID}
    USERMAP_GID=${USERMAP_GID:-$USERMAP_ORIG_GID}
    USERMAP_GIDS=${USERMAP_GIDS:-$USERMAP_ORIG_GIDS}
    if [ "${USERMAP_UID}" != "${USERMAP_ORIG_UID}" ] || [ "${USERMAP_GID}" != "${USERMAP_ORIG_GID}" ] || [ "${USERMAP_GIDS}" != "${USERMAP_ORIG_GIDS}" ]; then
        echo "Starting with UID : $USERMAP_UID, GID: $USERMAP_GID, GIDs: $USERMAP_GIDS"
        usermod -u $USERMAP_UID -o go
        groupmod -g $USERMAP_GID go
        for i in $USERMAP_GIDS; do
            if [ $USERMAP_GID -ne $i ]; then
                groupadd -g $i $i;
                usermod -aG $i go
            fi
        done
        chown go:go /opt/go
    fi
}

if [ "$(id -u)" = '0' ]; then
    map_uidgid
    exec gosu go "$@"
else
    exec "$@"
fi
