#!/bin/bash

set -o errexit
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE}")/..

docker run --rm -v $REPO_ROOT:/home/app ghcr.io/rajatjindal/krew-release-bot:v0.0.46 krew-release-bot template --tag $1 --template-file .krew.yaml > $REPO_ROOT/plugins/clusterctl-operator.yaml