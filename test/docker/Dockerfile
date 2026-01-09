FROM debian:trixie
LABEL Description="Fioup testing dockerfile for Debian Trixie"

ENV DEBIAN_FRONTEND=noninteractive
ARG SRC_DIR=$PWD

# It is important to run these in the same RUN command, because otherwise
# Docker layer caching breaks us
RUN apt-get update && apt-get -y install --no-install-suggests --no-install-recommends \
  apt-transport-https \
  apt-utils \
  bash \
  curl \
  e2fslibs-dev \
  git \
  vim \
  jq \
  lshw \
  make \
  net-tools \
  psmisc \
  python-is-python3 \
  sqlite3 \
  wget \
  zip \
  docker-compose \
  sudo \
  shellcheck \
  python3-docker \
  golang \
  python3-pytest \
  skopeo \
  python3-yaml \
  python3-expandvars

RUN git config --global --add safe.directory .

# Add Docker's official GPG key:
RUN apt-get update && apt-get install -y ca-certificates curl
RUN install -m 0755 -d /etc/apt/keyrings
RUN curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
RUN chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

RUN apt-get update && apt-get -y install --no-install-suggests --no-install-recommends \
  docker-ce \
  docker-ce-cli

# Install gosu required for the entry/startup script to add a user and group in the container
RUN wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/1.17/gosu-amd64" && \
    chmod +x /usr/local/bin/gosu && \
    gosu nobody true

RUN mkdir -p /usr/share/fioconfig/handlers/
RUN mkdir -p /run/secrets && chmod a+rwx /run/secrets

ARG REG_CERT=registry/certs/registry.crt

RUN git clone https://github.com/foundriesio/ci-scripts.git /ci-scripts
ENV PYTHONPATH=/ci-scripts
ENV LAYERS_SIZE_SCRIPT=/ci-scripts/apps/get_layers_meta.py

COPY $REG_CERT /etc/ssl/certs/registry.crt
RUN cat /etc/ssl/certs/registry.crt >> /etc/ssl/certs/ca-certificates.crt

RUN ln -sfn ${SRC_DIR}/bin/fioup /usr/local/bin/fioup
RUN ln -sfn ${SRC_DIR}/bin/fioup /usr/local/bin/docker-credential-fioup
RUN ln -sfn ${SRC_DIR}/debian/fioconfig-handlers/aktualizr-toml-update /usr/share/fioconfig/handlers/aktualizr-toml-update

WORKDIR /build_composeapp
ENV STOREROOT=/var/sota/reset-apps
ENV COMPOSEROOT=/var/sota/compose-apps
RUN git clone https://github.com/foundriesio/composeapp.git && \
    cd composeapp && git checkout v95 && make && cp ./bin/composectl /usr/bin/
