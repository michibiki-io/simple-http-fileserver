#####################################
########## For Development ##########
#####################################
FROM golang:latest as development

ARG LIBVIPS_VERSION=8.9.2

# Installs libvips + required libraries
RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y \
  gosu sudo \
  ca-certificates \
  automake build-essential curl \
  gobject-introspection gtk-doc-tools libglib2.0-dev libexif-dev libxml2-dev \
  libjpeg62-turbo-dev libpng-dev libwebp-dev libtiff5-dev libgif-dev librsvg2-dev && \
  export LD_LIBRARY_PATH="/vips/lib:$LD_LIBRARY_PATH" && \
  export PKG_CONFIG_PATH="/vips/lib/pkgconfig:$PKG_CONFIG_PATH" && \
  cd /tmp && \
  curl -fsSLO https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.gz && \
  tar zvxf vips-${LIBVIPS_VERSION}.tar.gz && \
  cd /tmp/vips-${LIBVIPS_VERSION} && \
	CFLAGS="-g -O3" CXXFLAGS="-D_GLIBCXX_USE_CXX11_ABI=0 -g -O3" \
    ./configure \
    --disable-debug \
    --disable-dependency-tracking \
    --disable-introspection \
    --disable-static \
    --enable-gtk-doc-html=no \
    --enable-gtk-doc=no \
    --enable-pyvips8=no \
    --prefix=/opt/vips && \
  make && \
  make install

RUN mkdir -p /opt/go && \
    mkdir -p /opt/go/config && \
    mkdir -p /opt/vips && \
    groupadd go && useradd -m -s /bin/bash -g go go && \
    chown go:go /opt/go -R && \
    echo "Defaults:go !env_reset" > /etc/sudoers && \
    echo "go ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers && \
    echo '/opt/vips/lib' > /etc/ld.so.conf.d/vips.conf

# Envs
ENV LD_LIBRARY_PATH="/opt/vips/lib:$LD_LIBRARY_PATH" \
    PKG_CONFIG_PATH="/opt/vips/lib/pkgconfig:/usr/local/lib/pkgconfig:/usr/lib/pkgconfig:/usr/X11/lib/pkgconfig" \
    CONTEXT_PATH="" \
    AUTH_SERVER_AUTH_URL=http://localhost:80/v1/authorize \
    AUTH_SERVER_VERIFY_URL=http://localhost:80/v1/verify \
    AUTH_SERVER_REFRESH_URL=http://localhost:80/v1/refresh \
    AUTH_SERVER_DEAUTH_URL=http://localhost:80/v1/deauthorize \
    AUTH_SERVER_ACCESS_TOKEN_JSON_PATH=access_token \
    AUTH_SERVER_REFRESH_TOKEN_JSON_PATH=refresh_token \
    AUTH_SERVER_EXPIREIN_JSON_PATH=expire_in \
    AUTH_SERVER_USERID_JSON_PATH=user.Id \
    AUTH_SERVER_GROUP_JSON_PATH=user.Groups \
    SESSION_BACKEND=cookie \
    SESSION_BACKEND_HOST=localhost:6379 \
    SESSION_STORE_USER_MAX_AGE=300 \
    API_TOKEN_EXPIRE_IN=600

RUN ldconfig

WORKDIR /opt/go

COPY . .

# require tools
RUN GOBIN=/tmp/ go get github.com/go-delve/delve/cmd/dlv@master && \
    mv /tmp/dlv $GOPATH/bin/dlv-dap && \
    go install golang.org/x/tools/gopls@latest && \
    go mod tidy && go build -o ./simple ./main.go

COPY entrypoint.sh /opt
RUN chmod go+x /opt/entrypoint.sh

VOLUME ["/opt/go", "/public", "/private"]

EXPOSE 8080

ENTRYPOINT ["/opt/entrypoint.sh"]

CMD ["/bin/bash"]

#################################
########## For RUNTIME ##########
#################################
FROM debian:bullseye-slim as runtime

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gosu sudo \
    libglib2.0-0 \
    libexif12 \
    libxml2 \
    libjpeg62-turbo \
    libpng16-16 \
    libgif7 \
    libwebp6 \
    libwebpmux3 \
    libwebpdemux2 \
    libtiff5 \
    librsvg2-2 && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/go && \
    mkdir -p /opt/go/config && \
    mkdir -p /opt/vips

COPY --from=development /opt/go/simple /opt/go/simple
COPY --from=development /opt/vips /opt/vips
COPY ./static /opt/go/static
COPY ./templates /opt/go/templates

# Envs
ENV GIN_MODE=release \ 
    CONTEXT_PATH="" \
    LD_LIBRARY_PATH="/opt/vips/lib:$LD_LIBRARY_PATH" \
    AUTH_SERVER_AUTH_URL=http://localhost:80/v1/authorize \
    AUTH_SERVER_VERIFY_URL=http://localhost:80/v1/verify \
    AUTH_SERVER_REFRESH_URL=http://localhost:80/v1/refresh \
    AUTH_SERVER_DEAUTH_URL=http://localhost:80/v1/deauthorize \
    AUTH_SERVER_ACCESS_TOKEN_JSON_PATH=access_token \
    AUTH_SERVER_REFRESH_TOKEN_JSON_PATH=refresh_token \
    AUTH_SERVER_EXPIREIN_JSON_PATH=expire_in \
    AUTH_SERVER_USERID_JSON_PATH=user.Id \
    AUTH_SERVER_GROUP_JSON_PATH=user.Groups \
    SESSION_BACKEND=cookie \
    SESSION_BACKEND_HOST=localhost:6379 \
    SESSION_STORE_USER_MAX_AGE=300 \
    API_TOKEN_EXPIRE_IN=600

RUN groupadd go && useradd -m -s /bin/bash -g go go && \
    chown go:go /opt/go -R && \
    echo '/opt/vips/lib' > /etc/ld.so.conf.d/vips.conf && \
    ldconfig

COPY entrypoint.sh /opt
RUN chmod go+x /opt/entrypoint.sh

WORKDIR /opt/go

EXPOSE 8080

VOLUME ["/opt/go/config", "/public", "/private"]

ENTRYPOINT ["/opt/entrypoint.sh"]

CMD ["/opt/go/simple"]
