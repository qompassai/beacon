#!/bin/sh

# Abort on error.
set -e

platforms=linux/amd64,linux/arm64,linux/arm,linux/386,linux/ppc64le,linux/s390x

# We are building by "go install github.com/qompassai/beacon@$beaconversion", to ensure the
# binary gets a proper version stamped into its buildinfo. It also helps to ensure
# there is no accidental local change in the image.
beaconversion=$(go list -mod mod -m github.com/qompassai/beacon@$(git rev-parse HEAD) | cut -f2 -d' ')
echo Building beacon $beaconversion for $platforms, without local/uncommitted changes

# Ensure latest golang and alpine docker images.
podman image pull --quiet docker.io/golang:1-alpine
for i in $(echo $platforms | sed 's/,/ /g'); do
    podman image pull --quiet --platform $i docker.io/alpine:latest
done
# "Last pulled" apparently is the one used for "podman run" below, not the one
# that matches the platform. So pull for current platform again.
podman image pull --quiet docker.io/alpine:latest

# Get the goland and alpine versions from the docker images.
goversion=$(podman run golang:1-alpine go version | cut -f3 -d' ')
alpineversion=alpine$(podman run alpine:latest cat /etc/alpine-release)
# We assume the alpines for all platforms have the same version...
echo Building with $goversion and $alpineversion

# We build the images individually so we can pass goos and goarch ourselves,
# needed because the platform in "FROM --platform <image>" in the first stage
# seems to override the TARGET* variables.
test -d empty || mkdir empty
((rm -r tmp/gomod || exit 0); mkdir -p tmp/gomod) # fetch modules through goproxy just once
(podman manifest rm beacon:$beaconversion-$goversion-$nixversion || exit 0)
for platform in $(echo $platforms | sed 's/,/ /g'); do
    goos=$(echo $platform | sed 's,/.*$,,')
    goarch=$(echo $platform | sed 's,^.*/,,')
    podman build --platform $platform -f Dockerfile.release -v $HOME/go/pkg/sumdb:/go/pkg/sumbd:ro -v $PWD/tmp/gomod:/go/pkg/mod --build-arg goos=$goos --build-arg goarch=$goarch --build-arg beaconversion=$beaconversion --manifest beacon:$beaconversion-$goversion-$nixversion empty
done

cat <<EOF

# Suggested commands to push images:

podman manifest push --all beacon:$beaconversion-$goversion-$nixversion \$host/beacon:$beaconversion-$goversion-$nixversion

podman manifest push --all beacon:$beaconversion-$goversion-$nixversion \$host/beacon:$beaconversion
podman manifest push --all beacon:$beaconversion-$goversion-$nixversion \$host/beacon:latest
EOF
