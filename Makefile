# Copyright 2021 The Kubernetes Authors.
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

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL:=/usr/bin/env bash

# Path to main repo
ROOT:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.DEFAULT_GOAL:=help

GO_VERSION ?= 1.23.0
GO_BASE_CONTAINER ?= docker.io/library/golang
GO_CONTAINER_IMAGE = $(GO_BASE_CONTAINER):$(GO_VERSION)

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq ($(GOPROXY),)
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Use GOPRIVATE environment variable if set
GOPRIVATE := $(shell go env GOPRIVATE)
export GOPRIVATE

# Base docker images

DOCKERFILE_CONTAINER_IMAGE ?= docker.io/docker/dockerfile:1.4
DEPLOYMENT_BASE_IMAGE ?= gcr.io/distroless/static
DEPLOYMENT_BASE_IMAGE_TAG ?= nonroot-${ARCH}

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE=on

BUILD_CONTAINER_ADDITIONAL_ARGS ?=

# This option is for running docker manifest command
export DOCKER_CLI_EXPERIMENTAL := enabled

CURL_RETRIES=3

# Directories
TOOLS_DIR := $(ROOT)/hack/tools
TEST_DIR := $(ROOT)/test
CHART_UPDATE_DIR := $(ROOT)/hack/chart-update
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
JUNIT_REPORT_DIR := $(TOOLS_DIR)/_out
BIN_DIR := bin
GO_INSTALL := ./scripts/go_install.sh

export PATH := $(abspath $(TOOLS_BIN_DIR)):$(PATH)

# Kubebuilder
export KUBEBUILDER_ENVTEST_KUBERNETES_VERSION ?= 1.30.3
export KUBEBUILDER_CONTROLPLANE_START_TIMEOUT ?= 60s
export KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT ?= 60s

# Release
USER_FORK ?= $(shell git config --get remote.origin.url | cut -d/ -f4) # only works on https://github.com/<username>/cluster-api.git style URLs
ifeq ($(USER_FORK),)
USER_FORK := $(shell git config --get remote.origin.url | cut -d: -f2 | cut -d/ -f1) # for git@github.com:<username>/cluster-api.git style URLs
endif
IMAGE_REVIEWERS ?= $(shell ./hack/get-project-maintainers.sh)

# Binaries.
# Need to use abspath so we can invoke these from subdirectories
CONTROLLER_GEN_VER := v0.16.1
CONTROLLER_GEN_BIN := controller-gen
CONTROLLER_GEN := $(TOOLS_BIN_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)

GOLANGCI_LINT_VER := v2.0.2
GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER)

KUSTOMIZE_VER := v5.3.0
KUSTOMIZE_BIN := kustomize
KUSTOMIZE := $(TOOLS_BIN_DIR)/$(KUSTOMIZE_BIN)-$(KUSTOMIZE_VER)

# This is a commit from CR main (22.05.2024).
# Intentionally using a commit from main to use a setup-envtest version
# that uses binaries from controller-tools, not GCS.
# CR PR: https://github.com/kubernetes-sigs/controller-runtime/pull/2811
SETUP_ENVTEST_VER := v0.0.0-20240522175850-2e9781e9fc60
SETUP_ENVTEST_BIN := setup-envtest
SETUP_ENVTEST := $(TOOLS_BIN_DIR)/$(SETUP_ENVTEST_BIN)-$(SETUP_ENVTEST_VER)

GOTESTSUM_VER := v1.11.0
GOTESTSUM_BIN := gotestsum
GOTESTSUM := $(TOOLS_BIN_DIR)/$(GOTESTSUM_BIN)-$(GOTESTSUM_VER)

GINKGO_VER := v2.22.2
GINKGO_BIN := ginkgo
GINKGO := $(TOOLS_BIN_DIR)/$(GINKGO_BIN)-$(GINKGO_VER)

ENVSUBST_VER := v2.0.0-20210730161058-179042472c46
ENVSUBST_BIN := envsubst
ENVSUBST := $(TOOLS_BIN_DIR)/$(ENVSUBST_BIN)-$(ENVSUBST_VER)

GO_APIDIFF_VER := v0.8.2
GO_APIDIFF_BIN := go-apidiff
GO_APIDIFF := $(TOOLS_BIN_DIR)/$(GO_APIDIFF_BIN)-$(GO_APIDIFF_VER)

HELM_VER := v3.14.4
HELM_BIN := helm
HELM := $(TOOLS_BIN_DIR)/$(HELM_BIN)-$(HELM_VER)

YQ_VER := v4.35.2
YQ_BIN := yq
YQ := $(TOOLS_BIN_DIR)/$(YQ_BIN)-$(YQ_VER)

KPROMO_VER := v4.0.5
KPROMO_BIN := kpromo
KPROMO :=  $(TOOLS_BIN_DIR)/$(KPROMO_BIN)-$(KPROMO_VER)

CONVERSION_GEN_VER := v0.29.2
CONVERSION_GEN_BIN := conversion-gen
CONVERSION_GEN := $(TOOLS_BIN_DIR)/$(CONVERSION_GEN_BIN)-$(CONVERSION_GEN_VER)

CONVERSION_VERIFIER_VER := v1.7.0
CONVERSION_VERIFIER_BIN := conversion-verifier
CONVERSION_VERIFIER := $(TOOLS_BIN_DIR)/$(CONVERSION_VERIFIER_BIN)-$(CONVERSION_VERIFIER_VER)

# It is set by Prow GIT_TAG, a git-based tag of the form vYYYYMMDD-hash, e.g., v20210120-v0.3.10-308-gc61521971
TAG ?= dev
ARCH ?= amd64
ALL_ARCH = amd64 arm arm64 ppc64le s390x

# Define Docker related variables. Releases should modify and double check these vars.
STAGING_REGISTRY ?= gcr.io/k8s-staging-capi-operator
STAGING_BUCKET ?= artifacts.k8s-staging-capi-operator.appspot.com

REGISTRY ?= $(STAGING_REGISTRY)
PROD_REGISTRY ?= registry.k8s.io/capi-operator

# Image name
IMAGE_NAME ?= cluster-api-operator
PACKAGE_NAME = cluster-api-operator
CONTROLLER_IMG ?= $(REGISTRY)/$(IMAGE_NAME)
CONTROLLER_IMG_TAG ?= $(CONTROLLER_IMG)-$(ARCH):$(TAG)

# Set build time variables including version details
LDFLAGS := $(shell $(ROOT)/hack/version.sh)

# Default cert-manager version
CERT_MANAGER_VERSION ?= v1.15.1

# E2E configuration
GINKGO_NOCOLOR ?= false
GINKGO_ARGS ?=
ARTIFACTS ?= $(ROOT)/_artifacts
E2E_CONF_FILE ?= $(ROOT)/test/e2e/config/operator-dev.yaml
E2E_CONF_FILE_ENVSUBST ?= $(ROOT)/test/e2e/config/operator-dev-envsubst.yaml
SKIP_CLEANUP ?= false
SKIP_CREATE_MGMT_CLUSTER ?= false
E2E_CERT_MANAGER_VERSION ?= $(CERT_MANAGER_VERSION)
E2E_OPERATOR_IMAGE ?= $(CONTROLLER_IMG):$(TAG)

# Relase
RELEASE_TAG ?= $(shell git describe --abbrev=0 2>/dev/null)
HELM_CHART_TAG := $(shell echo $(RELEASE_TAG) | cut -c 2-)
ifeq ($(HELM_CHART_TAG),)
	HELM_CHART_TAG := v0.0.1-test
	RELEASE_TAG := v0.0.1-test
endif
RELEASE_ALIAS_TAG ?= $(PULL_BASE_REF)
RELEASE_DIR := $(ROOT)/out
CHART_DIR := $(RELEASE_DIR)/charts/cluster-api-operator
CHART_PROVIDERS_DIR := $(RELEASE_DIR)/charts/cluster-api-operator-providers
CHART_PACKAGE_DIR := $(RELEASE_DIR)/package

# Set --output-base for conversion-gen if we are not within GOPATH
ROOT_DIR_RELATIVE := .
ifneq ($(abspath $(ROOT_DIR_RELATIVE)),$(shell go env GOPATH)/src/sigs.k8s.io/cluster-api-operator)
	CONVERSION_GEN_OUTPUT_BASE := --output-base=$(ROOT_DIR_RELATIVE)
else
	export GOPATH := $(shell go env GOPATH)
endif

all: generate test operator

help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9A-Za-z_-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Hack / Tools
## --------------------------------------

kustomize: $(KUSTOMIZE) ## Build a local copy of kustomize.
go-apidiff: $(GO_APIDIFF) ## Build a local copy of apidiff
ginkgo: $(GINKGO) ## Build a local copy of ginkgo
envsubst: $(ENVSUBST) ## Build a local copy of envsubst
controller-gen: $(CONTROLLER_GEN) ## Build a local copy of controller-gen.
setup-envtest: $(SETUP_ENVTEST) ## Build a local copy of setup-envtest.
golangci-lint: $(GOLANGCI_LINT) ## Build a local copy of golang ci-lint.
gotestsum: $(GOTESTSUM) ## Build a local copy of gotestsum.
helm: $(HELM) ## Build a local copy of helm.
yq: $(YQ) ## Build a local copy of yq.
kpromo: $(KPROMO) ## Build a local copy of kpromo.
conversion-gen: $(CONVERSION_GEN) ## Build a local copy of conversion-gen.
conversion-verifier: $(CONVERSION_VERIFIER) ## Build a local copy of conversion-verifier.

$(KUSTOMIZE): ## Build kustomize from tools folder.
	CGO_ENABLED=0 GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/kustomize/kustomize/v5 $(KUSTOMIZE_BIN) $(KUSTOMIZE_VER)

$(GO_APIDIFF): ## Build go-apidiff from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/joelanford/go-apidiff $(GO_APIDIFF_BIN) $(GO_APIDIFF_VER)

$(GINKGO): ## Build ginkgo from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/onsi/ginkgo/v2/ginkgo $(GINKGO_BIN) $(GINKGO_VER)

$(ENVSUBST): ## Build envsubst from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/drone/envsubst/v2/cmd/envsubst $(ENVSUBST_BIN) $(ENVSUBST_VER)

$(CONTROLLER_GEN): ## Build controller-gen from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/controller-gen $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)

$(SETUP_ENVTEST): # Build setup-envtest from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-runtime/tools/setup-envtest $(SETUP_ENVTEST_BIN) $(SETUP_ENVTEST_VER)

$(GOTESTSUM): # Build gotestsum from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) gotest.tools/gotestsum $(GOTESTSUM_BIN) $(GOTESTSUM_VER)

$(GOLANGCI_LINT): ## Build golangci-lint from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/golangci/golangci-lint/v2/cmd/golangci-lint $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VER)

$(HELM): ## Put helm into tools folder.
	mkdir -p $(TOOLS_BIN_DIR)
	rm -f "$(TOOLS_BIN_DIR)/$(HELM_BIN)*"
	curl --retry $(CURL_RETRIES) -fsSL -o $(TOOLS_BIN_DIR)/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
	chmod 700 $(TOOLS_BIN_DIR)/get_helm.sh
	USE_SUDO=false HELM_INSTALL_DIR=$(TOOLS_BIN_DIR) DESIRED_VERSION=$(HELM_VER) BINARY_NAME=$(HELM_BIN)-$(HELM_VER) $(TOOLS_BIN_DIR)/get_helm.sh
	ln -sf $(HELM) $(TOOLS_BIN_DIR)/$(HELM_BIN)
	rm -f $(TOOLS_BIN_DIR)/get_helm.sh

$(YQ):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) github.com/mikefarah/yq/v4 $(YQ_BIN) ${YQ_VER}

$(KPROMO):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) sigs.k8s.io/promo-tools/v4/cmd/kpromo $(KPROMO_BIN) ${KPROMO_VER}

$(CONVERSION_GEN):
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) k8s.io/code-generator/cmd/conversion-gen $(CONVERSION_GEN_BIN) ${CONVERSION_GEN_VER}

$(CONVERSION_VERIFIER):
	cd hack/tools/; GOBIN=$(TOOLS_BIN_DIR) go build -tags=tools -o $@ sigs.k8s.io/cluster-api/hack/tools/conversion-verifier

.PHONY: cert-mananger
cert-manager: # Install cert-manager on the cluster. This is used for development purposes only.
	$(ROOT)/hack/cert-manager.sh

## --------------------------------------
## Testing
## --------------------------------------

ARTIFACTS ?= ${ROOT}/_artifacts

KUBEBUILDER_ASSETS ?= $(shell $(SETUP_ENVTEST) use --use-env -p path $(KUBEBUILDER_ENVTEST_KUBERNETES_VERSION))

.PHONY: test
test: $(SETUP_ENVTEST) ## Run unit and integration tests
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test ./... $(TEST_ARGS)

.PHONY: test-verbose
test-verbose: ## Run tests with verbose settings.
	TEST_ARGS="$(TEST_ARGS) -v" $(MAKE) test

.PHONY: test-junit
test-junit: $(SETUP_ENVTEST) $(GOTESTSUM) ## Run tests with verbose setting and generate a junit report
	mkdir -p $(ARTIFACTS)
	set +o errexit; (KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test -json ./... $(TEST_ARGS); echo $$? > $(ARTIFACTS)/junit.exitcode) | tee $(ARTIFACTS)/junit.stdout
	$(GOTESTSUM) --junitfile $(ARTIFACTS)/junit.xml --raw-command cat $(ARTIFACTS)/junit.stdout
	exit $$(cat $(ARTIFACTS)/junit.exitcode)

## --------------------------------------
## Binaries
## --------------------------------------

.PHONY: operator
operator: ## Build operator binary
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/operator cmd/main.go

.PHONY: plugin
plugin: ## Build plugin binary
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/clusterctl-operator cmd/plugin/main.go

## --------------------------------------
## Lint / Verify
## --------------------------------------

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint the codebase
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_EXTRA_ARGS) --timeout=10m
	cd $(TEST_DIR); $(GOLANGCI_LINT) run --path-prefix $(TEST_DIR) --build-tags e2e -v $(GOLANGCI_LINT_EXTRA_ARGS) --timeout=10m

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Lint the codebase and run auto-fixers if supported by the linter
	GOLANGCI_LINT_EXTRA_ARGS=--fix $(MAKE) lint

.PHONY: apidiff
apidiff: $(GO_APIDIFF) ## Check for API differences
	$(GO_APIDIFF) $(shell git rev-parse origin/main) --print-compatible

.PHONY: verify
verify:
	$(MAKE) verify-modules
	$(MAKE) verify-gen

.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod $(TOOLS_DIR)/go.mod $(TOOLS_DIR)/go.sum $(CHART_UPDATE_DIR)/go.mod $(CHART_UPDATE_DIR)/go.sum $(TEST_DIR)/go.mod $(TEST_DIR)/go.sum); then \
		git diff; \
		echo "go module files are out of date"; exit 1; \
	fi

.PHONY: verify-gen
verify-gen: generate
	@if !(git diff --quiet HEAD); then \
		git diff; \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi

## --------------------------------------
## Generate / Manifests
## --------------------------------------

.PHONY: generate
generate: $(CONTROLLER_GEN) $(HELM) release-chart ## Generate code
	$(MAKE) generate-manifests
	$(MAKE) generate-go
	$(HELM) template capi-operator $(CHART_PACKAGE_DIR)/$(PACKAGE_NAME)-$(HELM_CHART_TAG).tgz > test/e2e/resources/full-chart-install.yaml

.PHONY: generate-go
generate-go: $(CONTROLLER_GEN) ## Runs Go related generate targets for the operator
	$(CONTROLLER_GEN) \
		object:headerFile=$(ROOT)/hack/boilerplate.go.txt \
		paths=./api/...

.PHONY: generate-manifests
generate-manifests: $(CONTROLLER_GEN) ## Generate manifests for the operator e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) \
		paths=./cmd \
		paths=./api/... \
		paths=./internal/controller/... \
		paths=./internal/webhook/... \
		crd:crdVersions=v1 \
		rbac:roleName=manager-role \
		output:crd:dir=./config/crd/bases \
		output:rbac:dir=./config/rbac \
		output:webhook:dir=./config/webhook \
		webhook

.PHONY: modules
modules: ## Runs go mod to ensure modules are up to date.
	go mod tidy
	cd $(TOOLS_DIR); go mod tidy
	cd $(CHART_UPDATE_DIR); go mod tidy
	cd $(TEST_DIR); go mod tidy

## --------------------------------------
## Docker
## --------------------------------------

.PHONY: docker-pull-prerequisites
docker-pull-prerequisites:
	docker pull $(DOCKERFILE_CONTAINER_IMAGE)
	docker pull $(GO_CONTAINER_IMAGE)
	docker pull $(DEPLOYMENT_BASE_IMAGE):$(DEPLOYMENT_BASE_IMAGE_TAG)

.PHONY: docker-build
docker-build: docker-pull-prerequisites ## Build the docker image for controller-manager
	docker build $(BUILD_CONTAINER_ADDITIONAL_ARGS) --build-arg builder_image=$(GO_CONTAINER_IMAGE) --build-arg deployment_base_image=$(DEPLOYMENT_BASE_IMAGE) --build-arg deployment_base_image_tag=$(DEPLOYMENT_BASE_IMAGE_TAG) --build-arg goproxy=$(GOPROXY) --build-arg goprivate=$(GOPRIVATE) --build-arg ARCH=$(ARCH) --build-arg LDFLAGS="$(LDFLAGS)" . -t $(CONTROLLER_IMG_TAG)

.PHONY: docker-push
docker-push: ## Push the docker image
	docker push $(CONTROLLER_IMG_TAG)

.PHONY: staging-manifests
staging-manifests:
	$(MAKE) manifest-modification PULL_POLICY=IfNotPresent RELEASE_TAG=$(RELEASE_ALIAS_TAG)
	$(MAKE) release-manifests

## --------------------------------------
## Docker — All ARCH
## --------------------------------------

.PHONY: docker-build-all ## Build all the architecture docker images
docker-build-all: $(addprefix docker-build-,$(ALL_ARCH))

docker-build-%:
	$(MAKE) ARCH=$* docker-build

.PHONY: docker-push-all ## Push all the architecture docker images
docker-push-all: $(addprefix docker-push-,$(ALL_ARCH))
	$(MAKE) docker-push-manifest

.PHONY: docker-push-manifest
docker-push-manifest: ## Push the fat manifest docker image.
	## Minimum docker version 18.06.0 is required for creating and pushing manifest images.
	docker manifest create --amend $(CONTROLLER_IMG):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(CONTROLLER_IMG)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${CONTROLLER_IMG}:${TAG} ${CONTROLLER_IMG}-$${arch}:${TAG}; done
	docker manifest push --purge ${CONTROLLER_IMG}:${TAG}

docker-push-%:
	$(MAKE) ARCH=$* docker-push

.PHONY: docker-build-e2e
docker-build-e2e:
	$(MAKE) CONTROLLER_IMG_TAG="$(E2E_OPERATOR_IMAGE)" docker-build

.PHONY: set-manifest-pull-policy
set-manifest-pull-policy:
	$(info Updating kustomize pull policy file for manager resources)
	sed -i'' -e 's@imagePullPolicy: .*@imagePullPolicy: '"$(PULL_POLICY)"'@' $(TARGET_RESOURCE)

.PHONY: set-manifest-pull-policy-chart
set-manifest-pull-policy-chart: $(YQ)
	$(info Updating image pull policy value for helm chart)
	$(YQ) eval '.image.manager.pullPolicy = "$(PULL_POLICY)"' $(TARGET_RESOURCE) -i

.PHONY: set-manifest-image
set-manifest-image:
	$(info Updating kustomize image patch file for manager resource)
	sed -i'' -e 's@image: .*@image: '"${MANIFEST_IMG}:$(MANIFEST_TAG)"'@' $(TARGET_RESOURCE)

.PHONY: set-manifest-image-chart
set-manifest-image-chart: $(YQ)
	$(info Updating image URL and tag values for helm chart)
	$(YQ) eval '.image.manager.repository = "$(MANIFEST_IMG)"' $(TARGET_RESOURCE) -i
	$(YQ) eval '.image.manager.tag = "$(MANIFEST_TAG)"' $(TARGET_RESOURCE) -i

## --------------------------------------
## Release
## --------------------------------------

$(RELEASE_DIR):
	mkdir -p $(RELEASE_DIR)/

$(CHART_DIR):
	mkdir -p $(CHART_DIR)/templates

$(CHART_PACKAGE_DIR):
	mkdir -p $(CHART_PACKAGE_DIR)

$(CHART_PROVIDERS_DIR):
	mkdir -p $(CHART_PROVIDERS_DIR)/templates

.PHONY: release
release: clean-release $(RELEASE_DIR)  ## Builds and push container images using the latest git tag for the commit.
	@if [ -z "${RELEASE_TAG}" ]; then echo "RELEASE_TAG is not set"; exit 1; fi
	@if ! [ -z "$$(git status --porcelain)" ]; then echo "Your local git repository contains uncommitted changes, use git clean before proceeding."; exit 1; fi
	git checkout "${RELEASE_TAG}"
	# Set the manifest image to the production bucket.
	$(MAKE) manifest-modification REGISTRY=$(PROD_REGISTRY)
	$(MAKE) chart-manifest-modification REGISTRY=$(PROD_REGISTRY)
	$(MAKE) release-manifests
	$(MAKE) release-chart

.PHONY: manifest-modification
manifest-modification: # Set the manifest images to the staging/production bucket.
	$(MAKE) set-manifest-image \
		MANIFEST_IMG=$(REGISTRY)/$(IMAGE_NAME) MANIFEST_TAG=$(RELEASE_TAG) \
		TARGET_RESOURCE="./config/default/manager_image_patch.yaml"
	$(MAKE) set-manifest-pull-policy PULL_POLICY=IfNotPresent TARGET_RESOURCE="./config/default/manager_pull_policy.yaml"

.PHONY: chart-manifest-modification
chart-manifest-modification: # Set the manifest images to the staging/production bucket.
	$(MAKE) set-manifest-image-chart \
		MANIFEST_IMG=$(REGISTRY)/$(IMAGE_NAME) MANIFEST_TAG=$(RELEASE_TAG) \
		TARGET_RESOURCE="$(ROOT)/hack/charts/cluster-api-operator/values.yaml"
	$(MAKE) set-manifest-pull-policy-chart PULL_POLICY=IfNotPresent TARGET_RESOURCE="$(ROOT)/hack/charts/cluster-api-operator/values.yaml"

.PHONY: release-manifests
release-manifests: $(KUSTOMIZE) $(RELEASE_DIR) ## Builds the manifests to publish with a release
	$(KUSTOMIZE) build ./config/default > $(RELEASE_DIR)/operator-components.yaml

.PHONY: release-chart
release-chart: $(HELM) $(KUSTOMIZE) $(RELEASE_DIR) $(CHART_DIR) $(CHART_PROVIDERS_DIR) $(CHART_PACKAGE_DIR) ## Builds the chart to publish with a release
	# cluster-api-operator チャートの処理
	cp -rf $(ROOT)/hack/charts/cluster-api-operator/. $(CHART_DIR)
	$(KUSTOMIZE) build ./config/chart > $(CHART_DIR)/templates/operator-components.yaml
	$(HELM) package $(CHART_DIR) --app-version=$(HELM_CHART_TAG) --version=$(HELM_CHART_TAG) --destination=$(CHART_PACKAGE_DIR)

	# cluster-api-operator-providers チャートの処理
	cp -rf $(ROOT)/hack/charts/cluster-api-operator-providers/. $(CHART_PROVIDERS_DIR)
	$(HELM) package $(CHART_PROVIDERS_DIR) --app-version=$(HELM_CHART_TAG) --version=$(HELM_CHART_TAG) --destination=$(CHART_PACKAGE_DIR)

.PHONY: release-staging
release-staging: ## Builds and push container images and manifests to the staging bucket.
	$(MAKE) docker-build-all
	$(MAKE) docker-push-all
	$(MAKE) release-alias-tag
	$(MAKE) staging-manifests
	$(MAKE) upload-staging-artifacts

.PHONY: release-alias-tag
release-alias-tag: # Adds the tag to the last build tag.
	gcloud container images add-tag -q $(CONTROLLER_IMG):$(TAG) $(CONTROLLER_IMG):$(RELEASE_ALIAS_TAG)

.PHONY: upload-staging-artifacts
upload-staging-artifacts: ## Upload release artifacts to the staging bucket
	gsutil cp $(RELEASE_DIR)/* gs://$(STAGING_BUCKET)/components/$(RELEASE_ALIAS_TAG)/

.PHONY: update-helm-plugin-repo
update-helm-plugin-repo:
	./hack/update-plugin-yaml.sh $(RELEASE_TAG)
	./hack/update-helm-repo.sh $(RELEASE_TAG)
	./hack/publish-index-changes.sh $(RELEASE_TAG)

.PHONY: promote-images
promote-images: $(KPROMO)
	$(KPROMO) pr --project capi-operator --tag $(RELEASE_TAG) --reviewers "$(IMAGE_REVIEWERS)" --fork $(USER_FORK) --image cluster-api-operator --use-ssh=false

## --------------------------------------
## Cleanup / Verification
## --------------------------------------

.PHONY: verify-conversions
verify-conversions: $(CONVERSION_VERIFIER) ## Verifies expected API conversion are in place
	$(CONVERSION_VERIFIER)

.PHONY: clean-generated-conversions
clean-generated-conversions: ## Remove files generated by conversion-gen from the mentioned dirs
	(IFS=','; for i in $(SRC_DIRS); do find $$i -type f -name 'zz_generated.conversion*' -exec rm -f {} \;; done)

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin

.PHONY: clean-bin
clean-bin: ## Remove all generated binaries
	rm -rf bin
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-release
clean-release: ## Remove the release folder
	rm -rf $(RELEASE_DIR)

## --------------------------------------
## E2E
## --------------------------------------

.PHONY: test-e2e-local ## Run e2e tests locally
test-e2e-local: docker-build-e2e test-e2e

.PHONY: test-e2e
test-e2e: $(KUSTOMIZE)
	$(MAKE) release-manifests
	$(MAKE) release-chart
	$(MAKE) test-e2e-run

.PHONY: test-e2e-run
test-e2e-run: $(GINKGO) $(ENVSUBST) $(HELM) ## Run e2e tests
	E2E_OPERATOR_IMAGE=$(E2E_OPERATOR_IMAGE) E2E_CERT_MANAGER_VERSION=$(E2E_CERT_MANAGER_VERSION) $(ENVSUBST) < $(E2E_CONF_FILE) > $(E2E_CONF_FILE_ENVSUBST) && \
	$(GINKGO) -v -trace -tags=e2e --junit-report=junit_cluster_api_operator_e2e.xml --output-dir="${JUNIT_REPORT_DIR}" --no-color=$(GINKGO_NOCOLOR) $(GINKGO_ARGS) ./test/e2e -- \
		-e2e.artifacts-folder="$(ARTIFACTS)" \
		-e2e.config="$(E2E_CONF_FILE_ENVSUBST)"  -e2e.components=$(RELEASE_DIR)/operator-components.yaml \
		-e2e.skip-resource-cleanup=$(SKIP_CLEANUP) -e2e.use-existing-cluster=$(SKIP_CREATE_MGMT_CLUSTER) \
		-e2e.helm-binary-path=$(HELM) -e2e.chart-path=$(CHART_PACKAGE_DIR)/cluster-api-operator-$(HELM_CHART_TAG).tgz $(E2E_ARGS)

go-version: ## Print the go version we use to compile our binaries and images
	@echo $(GO_VERSION)
