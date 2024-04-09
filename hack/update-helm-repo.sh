#!/bin/bash

set -o errexit
set -o pipefail

# Resolve the absolute path of the directory containing the script
SCRIPT_DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
REPO_ROOT="$SCRIPT_DIR/.."

cd $REPO_ROOT/hack/chart-update; go run . -release-tag=$1; cd -
