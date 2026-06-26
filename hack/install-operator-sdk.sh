#!/usr/bin/env bash

# Install operator-sdk from GitHub releases
# Usage: install-operator-sdk.sh <target-path> <version>

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <target-path> <version>" >&2
    echo "Example: $0 ./bin/operator-sdk v1.38.0" >&2
    exit 1
fi

TARGET_PATH="$1"
VERSION="$2"

# Check if correct version is already installed
if [ -x "$TARGET_PATH" ]; then
    # Run version command from a temp directory to avoid PROJECT file issues
    INSTALLED_VERSION=$(cd /tmp && "$TARGET_PATH" version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "")
    if [ "$INSTALLED_VERSION" = "$VERSION" ]; then
        exit 0
    fi
    echo "$TARGET_PATH version $INSTALLED_VERSION does not match expected $VERSION. Removing and reinstalling."
    rm -f "$TARGET_PATH"
fi

# Download the binary
mkdir -p "$(dirname "$TARGET_PATH")"
OS=$(go env GOOS)
ARCH=$(go env GOARCH)
BINARY_NAME="operator-sdk_${OS}_${ARCH}"
DOWNLOAD_URL="https://github.com/operator-framework/operator-sdk/releases/download/${VERSION}/${BINARY_NAME}"

echo "Downloading ${BINARY_NAME} from operator-framework/operator-sdk ${VERSION}..."
curl -sSLo "$TARGET_PATH" "$DOWNLOAD_URL"
chmod +x "$TARGET_PATH"

echo "Installed operator-sdk $VERSION to $TARGET_PATH"
