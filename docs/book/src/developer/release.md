# Releasing New Versions

This document describes the release process for the Cluster API Operator.

1. Create a new release branch and cut a release tag.

```bash
git checkout -b release-0.1
git push -u upstream release-0.1
```

```bash
# Export the tag of the release to be cut, e.g.:
export RELEASE_TAG=v0.1.1

# Create tags locally
# Warning: The test tag MUST NOT be an annotated tag.
git tag -s -a ${RELEASE_TAG} -m ${RELEASE_TAG}
git tag test/${RELEASE_TAG}

# Push tags
# Note: `upstream` must be the remote pointing to `github.com/kubernetes-sigs/cluster-api-operator`.
git push upstream ${RELEASE_TAG}
git push upstream test/${RELEASE_TAG}
```

**Note:** You may encounter an ioctl error during tagging. To resolve this, you need to set the GPG_TTY environment variable as `export GPG_TTY=$(tty)`.

This will trigger a [release GitHub action](https://github.com/kubernetes-sigs/cluster-api-operator/blob/main/.github/workflows/release.yaml) that creates a release with operator components and the Helm chart. Concurrently, a Prow job will start to publish operator images to the staging registry.

2. Wait for the images to appear in the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-capi-operator/global/cluster-api-operator).

3. Create a GitHub [Personal access token](https://github.com/settings/tokens) if you don't already have one. We're going to use this for opening a PR to promote the images from staging to production.

```bash
export GITHUB_TOKEN=<your GH token>
export USER_FORK=<your GH account name>
make promote-images
```

After it has been tested, merge the PR and verify that the image is present in the production registry.

```bash
docker pull registry.k8s.io/capi-operator/cluster-api-operator:${RELEASE_TAG}
```

4. Switch back to the main branch and update `index.yaml` and `clusterctl-operator.yaml`. These are the sources for the operator Helm chart repository and the local krew plugin manifest index, respectively.

```bash
git checkout main
make update-helm-plugin-repo
```

5. Create a PR with the changes.

## Setup jobs and dashboards for a new release branch

The goal of this task is to have test coverage for the new release branch and results in testgrid.
We are currently running CI jobs only in main and latest stable release branch (i.e release-0.5 will be used as an example below) and all configurations are hosted in test-infra [repo](https://github.com/kubernetes/test-infra).

1. Create new jobs based on the jobs running against our `main` branch:
    1. Copy `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-periodics-main.yaml` to `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-periodics-release-0-5.yaml`.
    2. Copy `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-presubmits-main.yaml` to `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-presubmits-release-0-5.yaml`.
    3. Modify the following:
        1. Rename the jobs, e.g.: `periodic-cluster-api-operator-test-main` => `periodic-cluster-api-operator-test-release-0-5`.
        2. Change `annotations.testgrid-dashboards` to `sig-cluster-lifecycle-cluster-api-operator-0.5`.
        3. Change `annotations.testgrid-tab-name`, e.g. `capi-operator-test-main` => `capi-operator-test-release-0-5`.
        4. For periodics additionally:
            * Change `extra_refs[].base_ref` to `release-0.5` (for repo: `cluster-api-operator`).
        5. For presubmits additionally: Adjust branches: `^main$` => `^release-0.5$`.
2. Create a new dashboard for the new branch in: `test-infra/config/testgrids/kubernetes/sig-cluster-lifecycle/config.yaml` (`dashboard_groups` and `dashboards`).
    * Add a new entry `sig-cluster-lifecycle-cluster-api-operator-0.5` in both `dashboard_groups` and `dashboards` lists.
3. Remove tests for previous release branch.
    * For example, let's assume we just created tests for v0.5, then we can now drop test coverage for the release-0.4 branch.
4. Verify the jobs and dashboards a day later by taking a look at: `https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator-0.5`.

Prior art: https://github.com/kubernetes/test-infra/pull/30372