#!/usr/bin/env bash
set -euo pipefail

MOD_VERSION=$(go mod edit -json | jq -r .Go)
echo "go.mod version: $MOD_VERSION"

for wf in .github/workflows/*.yml; do
    echo "Checking $wf ..."

    WF_VERSIONS=$(yq -r '.jobs[].steps[] | select(.with["go-version"]) | .with["go-version"]' -o=tsv "$wf" | grep -v '^---$' || true)
    if [ -z "$WF_VERSIONS" ]; then
        echo "ℹ️ No go-version found in $wf (skipped)"
        continue
    fi

    UNIQUE_WF_VERSIONS=$(echo "$WF_VERSIONS" | sort -u)
    if [ "$(echo "$UNIQUE_WF_VERSIONS" | wc -l)" -ne 1 ]; then
        echo "❌ Multiple Go versions found in $wf:"
        echo "$UNIQUE_WF_VERSIONS"
        exit 1
    fi

    # At this point there's only one unique Go version
    if [ "$UNIQUE_WF_VERSIONS" != "$MOD_VERSION" ]; then
        echo "❌ Mismatch in $wf: go.mod=$MOD_VERSION but workflow uses $UNIQUE_WF_VERSIONS"
        exit 1
    fi

    echo "✅ $wf matches go.mod ($MOD_VERSION)"
done

echo "✅ All workflows consistent with go.mod ($MOD_VERSION)"
