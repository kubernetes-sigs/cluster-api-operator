#!/bin/bash

set -o errexit
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE}")/..

cd $REPO_ROOT/hack/chart-update; go run . -release-tag=$1; cd -
