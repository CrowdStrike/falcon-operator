#!/usr/bin/env bash

# Install envtest (setup-envtest) using go install with versioned binary
# Usage: install-envtest.sh <target-path> <version>

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <target-path> <version>" >&2
    echo "Example: $0 ./bin/setup-envtest release-0.23" >&2
    exit 1
fi

TARGET_PATH="$1"
VERSION="$2"
LOCALBIN="$(dirname "$TARGET_PATH")"
VERSIONED_BINARY="${TARGET_PATH}-${VERSION}"

# Check if versioned binary exists and symlink points to it
if [ -f "$VERSIONED_BINARY" ] && [ "$(readlink -- "$TARGET_PATH" 2>/dev/null)" = "$VERSIONED_BINARY" ]; then
    exit 0
fi

# Install the tool
echo "Downloading sigs.k8s.io/controller-runtime/tools/setup-envtest@${VERSION}"
mkdir -p "$LOCALBIN"
rm -f "$TARGET_PATH"

GOBIN="$LOCALBIN" go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@${VERSION}"

# Move to versioned name and create symlink
BINARY_NAME="$(basename "$TARGET_PATH")"
mv "$LOCALBIN/$BINARY_NAME" "$VERSIONED_BINARY"
ln -sf "$(realpath "$VERSIONED_BINARY")" "$TARGET_PATH"

echo "Installed setup-envtest $VERSION to $TARGET_PATH"
