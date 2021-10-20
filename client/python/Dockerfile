FROM ubuntu:18.04

LABEL maintainer "Satoshi Mohri <staratlas@mohri.info>"

ENV container docker
ENV DEBIAN_FRONTEND noninteractive

# Shift timezone to Asia/Tokyo and Set local to jp.
RUN apt-get update \
 && apt-get install -y apt-utils \
 && apt-get install -y tzdata \
 && apt-get install -y language-pack-ja software-properties-common \
 && update-locale LANG=ja_JP.UTF-8 \
 && apt-get clean \
 && apt-get autoremove -y \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*
ENV TZ Asia/Tokyo
ENV LANG ja_JP.UTF-8
ENV LANGUAGE ja_JP:ja
ENV LC_ALL ja_JP.UTF-8

# BASIC SETUP
RUN apt-get -y update && apt-get -y upgrade \
 && apt-get install -y sudo vim git wget curl tmux htop tk-dev \
 python3.7 python3.7-dev python3.7-tk python3.7-distutils python3.7-venv \
 libopencv-dev gosu libgtk2.0-0 canberra-gtk* \
 && apt-get autoremove -y && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

# INSTALL PIP
RUN curl -kL https://bootstrap.pypa.io/get-pip.py | python3.7
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3.7 10 && \
    update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.7 10

# INSTALL PYTHON PACKAGES
WORKDIR /project

# install pip package
RUN pip install --no-cache-dir \
    numpy \
    matplotlib \
    ipykernel \
    opencv-python \
    opencv-contrib-python \
    pandas \
    Pillow \
    jupyter \
    requests

# entrypoint
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

VOLUME ["/project"]

CMD ["/bin/bash"]
