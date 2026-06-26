#!/usr/bin/env bash

# Install kubebuilder from GitHub releases
# Usage: install-kubebuilder.sh <target-path> <version>

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <target-path> <version>" >&2
    echo "Example: $0 ./bin/kubebuilder v4.15.0" >&2
    exit 1
fi

TARGET_PATH="$1"
VERSION="$2"

# Check if correct version is already installed
if [ -x "$TARGET_PATH" ]; then
    INSTALLED_VERSION=$("$TARGET_PATH" version 2>&1 | grep -E '(KubeBuilderVersion|KubeBuilder)' | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/^v//')
    EXPECTED_VERSION=$(echo "$VERSION" | sed 's/^v//')
    if [ "$INSTALLED_VERSION" = "$EXPECTED_VERSION" ]; then
        exit 0
    fi
    echo "$TARGET_PATH version v$INSTALLED_VERSION does not match expected $VERSION. Removing and reinstalling."
    rm -f "$TARGET_PATH"
fi

# Download the binary
mkdir -p "$(dirname "$TARGET_PATH")"
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
BINARY_NAME="kubebuilder_${OS}_${ARCH}"
DOWNLOAD_URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/${VERSION}/${BINARY_NAME}"

echo "Downloading ${BINARY_NAME} from kubernetes-sigs/kubebuilder ${VERSION}..."
curl -sSLo "$TARGET_PATH" "$DOWNLOAD_URL"
chmod +x "$TARGET_PATH"

echo "Installed kubebuilder $VERSION to $TARGET_PATH"
"$TARGET_PATH" version 2>&1 | head -1
