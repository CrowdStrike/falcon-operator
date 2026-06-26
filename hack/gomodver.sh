#!/usr/bin/env bash

# gomodver.sh extracts the version of a Go module from go.mod
# Usage: gomodver.sh <module-path>
# Example: gomodver.sh sigs.k8s.io/controller-runtime
# Returns: v0.23.1 (or empty if not found)

set -e

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <module-path>" >&2
    echo "Example: $0 sigs.k8s.io/controller-runtime" >&2
    exit 1
fi

MODULE_PATH="$1"

# Query go.mod for the module version, handling replace directives
go list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' "$MODULE_PATH" 2>/dev/null || echo ""
