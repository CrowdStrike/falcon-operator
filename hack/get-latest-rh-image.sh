#!/usr/bin/env bash

set -e

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <component>" >&2
    echo "Available components: falcon-sensor, falcon-container, falcon-kac, falcon-imageanalyzer, falcon-operator" >&2
    exit 1
fi

COMPONENT="$1"

# Map component names to registry and repo paths
case "$COMPONENT" in
    falcon-sensor)
        REGISTRY="registry.crowdstrike.com"
        REPO_PATH="falcon-sensor/release/falcon-sensor"
        ;;
    falcon-container)
        REGISTRY="registry.crowdstrike.com"
        REPO_PATH="falcon-container/release/falcon-container"
        ;;
    falcon-kac)
        REGISTRY="registry.crowdstrike.com"
        REPO_PATH="falcon-kac/release/falcon-kac"
        ;;
    falcon-imageanalyzer)
        REGISTRY="registry.crowdstrike.com"
        REPO_PATH="falcon-imageanalyzer/release/falcon-imageanalyzer"
        ;;
    falcon-operator)
        REGISTRY="registry.connect.redhat.com"
        REPO_PATH="crowdstrike/falcon-operator"
        ;;
    *)
        echo "Error: Unknown component '$COMPONENT'" >&2
        echo "Available components: falcon-sensor, falcon-container, falcon-kac, falcon-imageanalyzer, falcon-operator" >&2
        exit 1
        ;;
esac

API_PREFIX="https://catalog.redhat.com/api/containers/v1/repositories/registry/${REGISTRY}/repository"

# First, get pagination info to determine the last page
FIRST_PAGE=$(curl -s "${API_PREFIX}/${REPO_PATH}/images")
TOTAL=$(echo "$FIRST_PAGE" | jq -r '.total')
PAGE_SIZE=$(echo "$FIRST_PAGE" | jq -r '.page_size')

# Calculate the last page number (pages are 0-indexed)
LAST_PAGE=$(( (TOTAL - 1) / PAGE_SIZE ))

# Fetch the last page
RESPONSE=$(curl -s "${API_PREFIX}/${REPO_PATH}/images?page=${LAST_PAGE}")

# Extract all repositories with their tags and manifest digests
# Filter to only numeric version tags, sort by version, and get the latest
RESULT=$(echo "$RESPONSE" | jq -r '
  .data[].repositories[] |
  select(.tags[0].name | test("^[0-9]")) |
  {
    tag: .tags[0].name,
    manifest_list_digest: .manifest_list_digest
  } |
  "\(.tag)|\(.manifest_list_digest)"
' | sort -V | tail -n1)

if [ -z "$RESULT" ]; then
    echo "Error: Could not retrieve image information from Red Hat catalog" >&2
    exit 1
fi

# Parse the result
TAG=$(echo "$RESULT" | cut -d'|' -f1)
DIGEST=$(echo "$RESULT" | cut -d'|' -f2)

echo "tag=${TAG}"
echo "manifest_list_digest=${DIGEST}"
