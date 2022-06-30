#!/bin/bash
set -o errexit
set -o pipefail

RELVERSION=${1//v/}

if [[ -z "$RELVERSION" ]]; then
    echo "usage: $0 RELEASE_VERSION"
    exit 1
fi

echo "Setting Release to version: $RELVERSION"
FILESDIRS="README.md deploy/parts/operator.yaml docs"

echo "Update docs for new release"
if [[ $OSTYPE == 'darwin'* ]]; then
  sed -i '' "s/^VERSION ?= .*/VERSION ?= $RELVERSION/g" Makefile
  LC_ALL=C find $FILESDIRS -type f -exec sed -i '' "/[c\|C]rowd[s\|S]trike\/falcon-operator/s/v\([0-9.]\+\)\{3\}/v$RELVERSION/g" {} +
else
  sed -i "s/^VERSION ?= .*/VERSION ?= $RELVERSION/g" Makefile
  find $FILESDIRS -type f -exec sed -i "/[c\|C]rowd[s\|S]trike\/falcon-operator/s/v\([0-9.]\+\)\{3\}/v$RELVERSION/g" {} \;
fi

echo "Update manifests and bundle for new Release"
make manifests bundle deploy/falcon-operator.yaml

