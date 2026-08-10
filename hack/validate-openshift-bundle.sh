#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CSV_FILE="${PROJECT_ROOT}/bundle/manifests/falcon-operator.clusterserviceversion.yaml"
ANNOTATIONS_FILE="${PROJECT_ROOT}/bundle/metadata/annotations.yaml"

PASS=0
FAIL=0

pass() {
    PASS=$((PASS + 1))
    echo "  ✓ $1"
}

fail() {
    FAIL=$((FAIL + 1))
    echo "  ✗ $1" >&2
}

check() {
    local desc="$1"
    local result="$2"
    if [ -n "$result" ] && [ "$result" != "null" ] && [ "$result" != "false" ]; then
        pass "$desc"
    else
        fail "$desc"
    fi
}

check_eq() {
    local desc="$1"
    local actual="$2"
    local expected="$3"
    if [ "$actual" = "$expected" ]; then
        pass "$desc"
    else
        fail "$desc (got: '$actual', expected: '$expected')"
    fi
}

echo "Validating OpenShift bundle: $CSV_FILE"
echo ""

if [ ! -f "$CSV_FILE" ]; then
    echo "Error: $CSV_FILE not found. Run 'make bundle-openshift' first." >&2
    exit 1
fi

# --- Section 1: Annotations ---
echo "Annotations:"

REPO=$(yq '.metadata.annotations.repository' "$CSV_FILE")
check_eq "repository annotation" "$REPO" "registry.connect.redhat.com/crowdstrike/falcon-operator"

SUPPORT=$(yq '.metadata.annotations.support' "$CSV_FILE")
check_eq "support annotation" "$SUPPORT" "CrowdStrike"

VALID_SUB=$(yq '.metadata.annotations."operators.openshift.io/valid-subscription"' "$CSV_FILE")
check_eq "valid-subscription annotation" "$VALID_SUB" "Not required to use this operator"

CONTAINER_IMAGE=$(yq '.metadata.annotations.containerImage' "$CSV_FILE")
if echo "$CONTAINER_IMAGE" | grep -q "registry.connect.redhat.com/crowdstrike/falcon-operator@sha256:"; then
    pass "containerImage uses RH registry with digest"
else
    fail "containerImage should use registry.connect.redhat.com with sha256 digest (got: $CONTAINER_IMAGE)"
fi

echo ""

# --- Section 2: Version and Replaces ---
echo "Version and Replaces:"

# Validate spec.version format
VERSION=$(yq '.spec.version' "$CSV_FILE")
if echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+'; then
    pass "spec.version is semver ($VERSION)"
else
    fail "spec.version is not valid semver (got: $VERSION)"
fi

# Validate spec.version matches Makefile VERSION
MAKEFILE_VERSION=$(grep -E '^VERSION\s*\?=' "${PROJECT_ROOT}/Makefile" | awk '{print $3}')
check_eq "spec.version matches Makefile VERSION" "$VERSION" "$MAKEFILE_VERSION"

# Validate spec.replaces exists and format
REPLACES=$(yq '.spec.replaces' "$CSV_FILE")
if [ -z "$REPLACES" ] || [ "$REPLACES" = "null" ]; then
    fail "spec.replaces is missing (bundle needs to be patched)"
elif echo "$REPLACES" | grep -qE '^falcon-operator\.v[0-9]+\.[0-9]+\.[0-9]+'; then
    pass "spec.replaces follows naming convention ($REPLACES)"

    # Extract version from replaces
    REPLACES_VERSION=$(echo "$REPLACES" | sed 's/^falcon-operator\.v//')

    # Validate replaces version differs from current version
    if [ "$VERSION" = "$REPLACES_VERSION" ]; then
        fail "spec.replaces version ($REPLACES_VERSION) must not match spec.version ($VERSION)"
    else
        pass "spec.replaces version differs from spec.version"
    fi

    # Validate replaces version matches Makefile PREVIOUS_VERSION
    MAKEFILE_PREVIOUS_VERSION=$(grep -E '^PREVIOUS_VERSION\s*\?=' "${PROJECT_ROOT}/Makefile" | awk '{print $3}')
    check_eq "spec.replaces matches Makefile PREVIOUS_VERSION" "$REPLACES_VERSION" "$MAKEFILE_PREVIOUS_VERSION"
else
    fail "spec.replaces should match falcon-operator.vX.Y.Z (got: $REPLACES)"
fi

echo ""

# --- Section 3: Deployment Image ---
echo "Deployment Image:"

MANAGER_IMAGE=$(yq '(.spec.install.spec.deployments[] | select(.name == "falcon-operator-controller-manager") | .spec.template.spec.containers[] | select(.name == "manager") | .image)' "$CSV_FILE")
if echo "$MANAGER_IMAGE" | grep -q "registry.connect.redhat.com/crowdstrike/falcon-operator@sha256:"; then
    pass "manager container uses RH registry with digest"
else
    fail "manager container image should use registry.connect.redhat.com with digest (got: $MANAGER_IMAGE)"
fi

echo ""

# --- Section 4: RELATED_IMAGE Environment Variables ---
echo "RELATED_IMAGE Environment Variables:"

EXPECTED_ENVS=("RELATED_IMAGE_NODE_SENSOR" "RELATED_IMAGE_SIDECAR_SENSOR" "RELATED_IMAGE_ADMISSION_CONTROLLER" "RELATED_IMAGE_IMAGE_ANALYZER")
EXPECTED_REPOS=("falcon-sensor/release/falcon-sensor" "falcon-container/release/falcon-container" "falcon-kac/release/falcon-kac" "falcon-imageanalyzer/release/falcon-imageanalyzer")

for i in "${!EXPECTED_ENVS[@]}"; do
    ENV_NAME="${EXPECTED_ENVS[$i]}"
    EXPECTED_REPO="${EXPECTED_REPOS[$i]}"

    ENV_VAL=$(yq "(.spec.install.spec.deployments[] | select(.name == \"falcon-operator-controller-manager\") | .spec.template.spec.containers[] | select(.name == \"manager\") | .env[] | select(.name == \"${ENV_NAME}\") | .value)" "$CSV_FILE")

    if [ -z "$ENV_VAL" ] || [ "$ENV_VAL" = "null" ]; then
        fail "$ENV_NAME not found"
    elif echo "$ENV_VAL" | grep -q "registry.crowdstrike.com/${EXPECTED_REPO}@sha256:"; then
        pass "$ENV_NAME present with digest"
    else
        fail "$ENV_NAME has unexpected value (got: $ENV_VAL)"
    fi
done

echo ""

# --- Section 5: Related Images ---
echo "Related Images:"

RELATED_COUNT=$(yq '.spec.relatedImages | length' "$CSV_FILE")
check_eq "relatedImages count" "$RELATED_COUNT" "5"

EXPECTED_NAMES=("operator" "node-sensor" "sidecar-sensor" "admission-controller" "image-analyzer")
EXPECTED_REGISTRIES=(
    "registry.connect.redhat.com/crowdstrike/falcon-operator"
    "registry.crowdstrike.com/falcon-sensor/release/falcon-sensor"
    "registry.crowdstrike.com/falcon-container/release/falcon-container"
    "registry.crowdstrike.com/falcon-kac/release/falcon-kac"
    "registry.crowdstrike.com/falcon-imageanalyzer/release/falcon-imageanalyzer"
)

for i in "${!EXPECTED_NAMES[@]}"; do
    name="${EXPECTED_NAMES[$i]}"
    expected_registry="${EXPECTED_REGISTRIES[$i]}"

    IMG=$(yq ".spec.relatedImages[] | select(.name == \"${name}\") | .image" "$CSV_FILE")
    if [ -z "$IMG" ] || [ "$IMG" = "null" ]; then
        fail "relatedImages entry '$name' not found"
    elif echo "$IMG" | grep -q "@sha256:"; then
        pass "relatedImages '$name' uses digest"

        # Validate the registry/repo path
        if echo "$IMG" | grep -q "^${expected_registry}@sha256:"; then
            pass "relatedImages '$name' uses correct registry"
        else
            fail "relatedImages '$name' should use ${expected_registry} (got: $IMG)"
        fi
    else
        fail "relatedImages '$name' should use digest (got: $IMG)"
    fi
done

echo ""

# --- Section 6: Bundle Metadata Annotations ---
echo "Bundle Metadata Annotations ($ANNOTATIONS_FILE):"

if [ ! -f "$ANNOTATIONS_FILE" ]; then
    fail "annotations.yaml not found"
else
    CHANNEL=$(yq '.annotations."operators.operatorframework.io.bundle.channels.v1"' "$ANNOTATIONS_FILE")
    check_eq "channel" "$CHANNEL" "certified-1.0"

    DEFAULT_CHANNEL=$(yq '.annotations."operators.operatorframework.io.bundle.channel.default.v1"' "$ANNOTATIONS_FILE")
    check_eq "default channel" "$DEFAULT_CHANNEL" "certified-1.0"

    OCP_VERSIONS=$(yq '.annotations."com.redhat.openshift.versions"' "$ANNOTATIONS_FILE")
    check_eq "OpenShift versions" "$OCP_VERSIONS" "v4.12"
fi

echo ""

# --- Summary ---
TOTAL=$((PASS + FAIL))
echo "==============================="
echo "Results: $PASS/$TOTAL passed, $FAIL failed"
echo "==============================="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
