#!/bin/sh -e
## Used to update the debian changelog (release notes) before a release
## is tagged.

if [ $# -ne 1 ] ; then
	echo "Usage: $0 <version>"
	echo "  example: $0 0.1.1"
	exit 0
fi
version=$1
version=${version#v} # make sure we don't do v0.1.1

sed -i 's/UNRELEASED; /released; /' debian/changelog

docker run --rm -i \
	-v `pwd`:`pwd` -w `pwd` \
	debian:trixie  <<EOF
set -ex

apt update && apt install -y git-buildpackage
useradd -m -u $(id -u) -s /bin/bash builder
sudo -u builder EMAIL=bot@foundries.io \
	gbp dch -N ${version} --ignore-branch
EOF

