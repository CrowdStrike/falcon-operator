#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

# returns the operator version in `Version-GitHash` format
add_git_commit_version() {
  local SEMVER=$1

  GIT_COMMIT=$(git rev-parse --short HEAD)
  # The commit hash retreived varies in length, so we will standardize to length received from the operator image
  GIT_COMMIT="${GIT_COMMIT:0:7}"
  VERSION="${SEMVER}-${GIT_COMMIT}"

  echo "$VERSION"
}

SEMVER=${1}

if [[ -z "$SEMVER" ]]; then
    echo "usage: $0 OPERATOR_VERSION"
    exit 1
fi

VERSION=$(add_git_commit_version "$SEMVER")

if [[ -d "bin/" ]] ; then
    EXE="bin/manager"
else
    EXE="manager"
fi

TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}

CGO_ENABLED=0 GOOS=${TARGETPLATFORM/\/*/} GOARCH=${TARGETPLATFORM/*\//} go build -a \
    -tags "exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp" \
    --ldflags="-X 'github.com/crowdstrike/falcon-operator/version.Version=${VERSION}'" \
    -o ${EXE} main.go
