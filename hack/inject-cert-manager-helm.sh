#!/bin/bash

# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script injects cert-manager dependency in the helm chart.
# Usage: ./inject-cert-manager-helm.sh <version>
# Example: ./inject-cert-manager-helm.sh v1.12.2

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE}")/..
CHART_DIR=${REPO_ROOT}/out/charts/cluster-api-operator

# Validate version input - matches "vX.Y.Z" (e.g. v1.0.0)
if [[ ! "$1" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    echo "Please provide a valid version in the semver format (e.g. v1.0.0)"
    exit 1
fi

VERSION=$1
URL="https://github.com/cert-manager/cert-manager/releases/download/${VERSION}/cert-manager.crds.yaml"
OUTPUT_DIR="${CHART_DIR}/crds"

# Create the output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Download and save the file
curl -L -o "${OUTPUT_DIR}/cert-manager.crds.yaml" "$URL"
echo "Downloaded cert-manager.crds.yaml for ${VERSION} and saved it in ${OUTPUT_DIR}"

# Modify version in Chart.yaml
CHART_FILE="${CHART_DIR}/Chart.yaml"
if [[ ! -f "$CHART_FILE" ]]; then
    echo "Chart.yaml not found in the chart folder."
    exit 2
fi

# Update cert-manager dependency version in Chart.yaml
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i "" "/dependencies:/,/alias: cert-manager/ s/\(^ *version: *\).*\$/\1$VERSION/" "$CHART_FILE"
else
    # Linux
    sed -i "/dependencies:/,/alias: cert-manager/ s/\(^ *version: *\).*\$/\1$VERSION/" "$CHART_FILE"
fi

# Fetch dependencies with Helm
helm dependency update ${CHART_DIR}

echo "Updated cert-manager dependency version in Chart.yaml to ${VERSION}"
