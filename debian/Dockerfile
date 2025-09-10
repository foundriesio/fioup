FROM debian:trixie as builder

RUN apt update -y && apt install -y build-essential devscripts golang

RUN useradd -m -u 1000 -s /bin/bash builder && mkdir /src && chown builder /src

COPY --chown=builder / /src/fioup

USER builder
RUN \
    cd /src/fioup && debuild

FROM scratch
COPY --from=builder /src/*.deb /
