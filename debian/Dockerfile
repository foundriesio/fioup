FROM debian:trixie AS builder

RUN apt update -y && apt install -y build-essential devscripts golang

RUN useradd -m -u 1000 -s /bin/bash builder && mkdir /src && chown builder /src

COPY --chown=builder / /src/fioup

USER builder
RUN \
    cd /src/fioup && debuild -us -uc

# The first 3 lines of dpkg-parsechange log are:
# * blank line
# * "fioup (0.1) UNRELEASED; urgency=medium" - not quite what we want
# * a line with a single "."
# drop those and build some release notes
RUN cd /src/fioup && \
    echo "Changes" > release-notes.txt && \
    dpkg-parsechangelog -l debian/changelog --count 1 -S Changes | tail -n+4 > release-notes.txt

FROM scratch
COPY --from=builder /src/*.deb /
COPY --from=builder /src/fioup/release-notes.txt /
