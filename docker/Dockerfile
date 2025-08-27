FROM debian:trixie
LABEL Description="Fioup testing dockerfile for Debian Trixie"

ENV DEBIAN_FRONTEND=noninteractive

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
  python3-pytest

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

# Install docker credential helper and auth configuration
COPY config.json /usr/lib/docker/config.json
COPY docker-credential-fioup-helper /usr/bin/docker-credential-fioup-helper

# Install gosu required for the entry/startup script to add a user and group in the container
RUN wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/1.17/gosu-amd64" && \
    chmod +x /usr/local/bin/gosu && \
    gosu nobody true

# Copy the entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["bash"]
