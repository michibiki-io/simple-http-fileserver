#!/bin/bash

docker run --rm -d --name python37-ubuntu1804 --tty=true -v $(pwd):/project -e LOCAL_UID=$(id -u) -e LOCAL_GID="$(id -G)" michibiki.io/python3.7-ubuntu18.04:latest /bin/bash
