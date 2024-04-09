#!/bin/bash

set -o errexit
set -o pipefail

# Resolve the absolute path of the directory containing the script
SCRIPT_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
REPO_ROOT="$SCRIPT_DIR/.."

docker run --rm -v "$REPO_ROOT":/home/app ghcr.io/rajatjindal/krew-release-bot:v0.0.46 krew-release-bot template --tag "$1" --template-file .krew.yaml > "$REPO_ROOT"/plugins/clusterctl-operator.yaml
