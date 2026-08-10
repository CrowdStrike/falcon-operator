#!/usr/bin/env bash
#
# Patch OpenShift bundle with certification requirements
#
# This script handles image digest management and applies OpenShift patches.
#
# Usage:
#   ./patch-openshift-bundle.sh          # Populate patches with real digests
#   ./patch-openshift-bundle.sh --apply   # Apply all patches to generated bundle
#   ./patch-openshift-bundle.sh --restore # Restore patches to placeholder templates
#
# Process:
# 1. Populate: Fetch all image digests and update patch templates
# 2. Apply: Merge all patch files onto the generated bundle CSV
# 3. Restore: Restore placeholders for version control
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
IMAGES_PATCH_FILE="${IMAGES_PATCH_FILE:-${PROJECT_ROOT}/config/manifests-openshift/patches/images.yaml}"
PATCHES_DIR="${PROJECT_ROOT}/config/manifests-openshift/patches"
BUNDLE_CSV="${PROJECT_ROOT}/bundle/manifests/falcon-operator.clusterserviceversion.yaml"
GET_IMAGE_SCRIPT="${SCRIPT_DIR}/get-latest-rh-image.sh"

# Check for --apply flag
if [ "$1" = "--apply" ]; then
    echo "Applying OpenShift patches to bundle CSV..."

    if [ ! -f "$BUNDLE_CSV" ]; then
        echo "Error: $BUNDLE_CSV not found" >&2
        exit 1
    fi

    # All patches are JSON 6902 format — no metadata.name injection needed.
    # Write to a temp file first so the bundle CSV is not truncated on failure.
    echo "  Running kustomize to apply patches..."
    TEMP_OUTPUT=$(mktemp)
    (cd "${PROJECT_ROOT}/config/manifests-openshift" && "${PROJECT_ROOT}/bin/kustomize" build --load-restrictor LoadRestrictionsNone . > "$TEMP_OUTPUT")
    mv "$TEMP_OUTPUT" "$BUNDLE_CSV"

    echo "✓ OpenShift patches applied to bundle"
    exit 0
fi

# Check for --restore flag
if [ "$1" = "--restore" ]; then
    echo "Restoring image digest placeholders..."

    if [ ! -f "$IMAGES_PATCH_FILE" ]; then
        echo "Error: $IMAGES_PATCH_FILE not found" >&2
        exit 1
    fi

    # Replace all sha256 digests with generic DIGEST placeholder first
    yq -i '(... | select(tag == "!!str")) |= sub("@sha256:[a-f0-9]{64}", "@DIGEST")' "$IMAGES_PATCH_FILE"

    # Then replace DIGEST with specific placeholders based on image name
    yq -i '(... | select(tag == "!!str" and test("falcon-operator@DIGEST"))) |= sub("@DIGEST", "@OPERATOR_DIGEST")' "$IMAGES_PATCH_FILE"
    yq -i '(... | select(tag == "!!str" and test("falcon-sensor@DIGEST"))) |= sub("@DIGEST", "@SENSOR_DIGEST")' "$IMAGES_PATCH_FILE"
    yq -i '(... | select(tag == "!!str" and test("falcon-container@DIGEST"))) |= sub("@DIGEST", "@CONTAINER_DIGEST")' "$IMAGES_PATCH_FILE"
    yq -i '(... | select(tag == "!!str" and test("falcon-kac@DIGEST"))) |= sub("@DIGEST", "@KAC_DIGEST")' "$IMAGES_PATCH_FILE"
    yq -i '(... | select(tag == "!!str" and test("falcon-imageanalyzer@DIGEST"))) |= sub("@DIGEST", "@IAR_DIGEST")' "$IMAGES_PATCH_FILE"

    echo "✓ Patch file placeholders restored"
    exit 0
fi

# Normal mode: populate patches with real digests
if [ ! -f "$IMAGES_PATCH_FILE" ]; then
    echo "Error: $IMAGES_PATCH_FILE not found" >&2
    exit 1
fi

if [ ! -f "$GET_IMAGE_SCRIPT" ]; then
    echo "Error: $GET_IMAGE_SCRIPT not found" >&2
    exit 1
fi

echo "Fetching latest image digests for OpenShift bundle..."

echo "  Fetching falcon-operator image..."
eval "$("$GET_IMAGE_SCRIPT" falcon-operator)"
OPERATOR_DIGEST="$manifest_list_digest"

echo "  Fetching falcon-sensor image..."
eval "$("$GET_IMAGE_SCRIPT" falcon-sensor)"
SENSOR_DIGEST="$manifest_list_digest"

echo "  Fetching falcon-container image..."
eval "$("$GET_IMAGE_SCRIPT" falcon-container)"
CONTAINER_DIGEST="$manifest_list_digest"

echo "  Fetching falcon-kac image..."
eval "$("$GET_IMAGE_SCRIPT" falcon-kac)"
KAC_DIGEST="$manifest_list_digest"

echo "  Fetching falcon-imageanalyzer image..."
eval "$("$GET_IMAGE_SCRIPT" falcon-imageanalyzer)"
IAR_DIGEST="$manifest_list_digest"

echo "Updating image patch file with fetched digests..."
yq -i "(... | select(tag == \"!!str\")) |= sub(\"OPERATOR_DIGEST\", \"${OPERATOR_DIGEST}\")" "$IMAGES_PATCH_FILE"
yq -i "(... | select(tag == \"!!str\")) |= sub(\"SENSOR_DIGEST\", \"${SENSOR_DIGEST}\")" "$IMAGES_PATCH_FILE"
yq -i "(... | select(tag == \"!!str\")) |= sub(\"CONTAINER_DIGEST\", \"${CONTAINER_DIGEST}\")" "$IMAGES_PATCH_FILE"
yq -i "(... | select(tag == \"!!str\")) |= sub(\"KAC_DIGEST\", \"${KAC_DIGEST}\")" "$IMAGES_PATCH_FILE"
yq -i "(... | select(tag == \"!!str\")) |= sub(\"IAR_DIGEST\", \"${IAR_DIGEST}\")" "$IMAGES_PATCH_FILE"

echo "✓ Image digests updated in kustomize patch"
