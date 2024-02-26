# CI Jobs

This document intends to provide an overview over our jobs running via Prow, GitHub actions and Google Cloud Build.
It also documents the cluster-api-operator specific configuration in test-infra.

## Builds and Tests running on the main branch

> NOTE: To see which test jobs execute which tests or e2e tests, you can click on the links which lead to the respective test overviews in testgrid.

The dashboards for the ProwJobs can be found here: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator

More details about ProwJob configurations can be found [here](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes-sigs/cluster-api-operator).

### Presubmits

Prow Presubmits:
* mandatory for merge, always run:
  * [pull-cluster-api-operator-build-main] `./scripts/ci-build.sh`
  * [pull-cluster-api-operator-make-main] `./scripts/ci-make.sh`
  * [pull-cluster-api-operator-verify-main] `./scripts/ci-verify.sh`
* mandatory for merge, run if go code changes:
  * [pull-cluster-api-operator-test-main] `./scripts/ci-test.sh`
  * [pull-cluster-api-operator-e2e-main] `./scripts/ci-e2e.sh`
* optional for merge, run if go code changes:
  * [pull-cluster-api-operator-apidiff-main] `./scripts/ci-apidiff.sh`

GitHub Presubmit Workflows:
* PR golangci-lint: golangci/golangci-lint-action
  * Runs golangci-lint. Can be run locally via `make lint`.
* PR verify: kubernetes-sigs/kubebuilder-release-tools verifier
  * Verifies the PR titles have a valid format, i.e. contains one of the valid icons.
  * Verifies the PR description is valid, i.e. is long enough.
* PR dependabot (run on dependabot PRs)
  * Regenerates Go modules and code.
  
Other Github workflows
* release (runs when tags are pushed)
  * Creates a GitHub release with release notes for the tag.
* book publishing
  * Deploys operator book to GitHub Pages

### Postsubmits

Prow Postsubmits:
* [post-cluster-api-operator-push-images] Google Cloud Build: `make release-staging`

### Periodics

Prow Periodics:
* [periodic-cluster-api-operator-test-main] `./scripts/ci-test.sh`
* [periodic-cluster-api-operator-e2e-main] `./scripts/ci-e2e.sh`

## Test-infra configuration

* config/jobs/image-pushing/k8s-staging-cluster-api.yaml
  * Configures postsubmit job to push images and manifests.
* config/jobs/kubernetes-sigs/cluster-api-operator/
  * Configures Cluster API Operator presubmit and periodic jobs.
* config/testgrids/kubernetes/sig-cluster-lifecycle/config.yaml
  * Configures Cluster API Operator testgrid dashboards.
* config/prow/plugins.yaml
  * `approve`: disable auto-approval of PR authors, ignore GitHub reviews (/approve is explicitly required)
  * `lgtm`: enables retaining lgtm through squash
  * `require_matching_label`: configures `needs-triage`
  * `plugins`: enables `require-matching-label` plugin
  * `external_plugins`: enables `cherrypicker` plugin
* label_sync/labels.yaml
  * Configures labels for the `cluster-api-operator` repository.

<!-- links -->
[pull-cluster-api-operator-build-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-build-main
[pull-cluster-api-operator-make-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-make-main
[pull-cluster-api-operator-verify-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-verify-main
[pull-cluster-api-operator-test-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-test-main
[pull-cluster-api-operator-e2e-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-e2e-main
[pull-cluster-api-operator-apidiff-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-pr-apidiff-main
[post-cluster-api-operator-push-images]: https://testgrid.k8s.io/sig-cluster-lifecycle-image-pushes#post-cluster-api-operator-push-images
[periodic-cluster-api-operator-test-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-test-main
[periodic-cluster-api-operator-e2e-main]: https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator#capi-operator-e2e-main