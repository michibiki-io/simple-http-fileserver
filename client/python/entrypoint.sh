#!/bin/bash

USER_ID=${LOCAL_UID:-9001}
GROUP_ID=${LOCAL_GID:-9001}
MAIN_GID=($GROUP_ID)

echo "Starting with UID : $USER_ID, GID: $GROUP_ID"
useradd -u $USER_ID -o -m mluser

for i in $GROUP_ID; do
    if [ $MAIN_GID -eq $i ]; then
        groupmod -g $i mluser
    else
        groupadd -g $i $i;
        usermod -aG $i mluser
    fi
done

chown mluser:mluser /home/mluser -R

export HOME=/home/mluser
exec /usr/sbin/gosu mluser "$@"
