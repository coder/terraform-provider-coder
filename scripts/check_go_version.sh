#!/usr/bin/env bash
set -euo pipefail

MOD_VERSION=$(go mod edit -json | jq -r .Go)
echo "go.mod version: $MOD_VERSION"
STATUS=0

if [[ " $* " == *" --fix "* ]]; then
    for wf in .github/workflows/*.{yml,yaml}; do
        sed -i "s/go-version:.*/go-version: \"${MOD_VERSION}\"/g" "${wf}"
    done
    exit 0
fi

for wf in .github/workflows/*.{yml,yaml}; do
    WF_VERSIONS=$(yq -r '.jobs[].steps[] | select(.with["go-version"]) | .with["go-version"]' -o=tsv "$wf" | grep -v '^---$' || true)
    if [[ -z "$WF_VERSIONS" ]]; then
        continue
    fi

    UNIQUE_WF_VERSIONS=$(sort -u <<<"$WF_VERSIONS")
    for ver in $UNIQUE_WF_VERSIONS; do
        if [[ $ver != "$MOD_VERSION" ]]; then
            STATUS=1
            echo "❌ $wf: go.mod=$MOD_VERSION but workflow uses $(tr '\n' ' ' <<<"$UNIQUE_WF_VERSIONS")"
            continue
        fi
    done
done

if [[ $STATUS  -eq 1 ]]; then
    echo "Re-run this script with --fix to automatically update workflows to match go.mod"
fi

exit $STATUS
