# Releasing New Versions

## Cut a release

This document describes the release process for the Cluster API Operator.

1. Clone the repository locally: 

```bash
git clone git@github.com:kubernetes-sigs/cluster-api-operator.git
```

2. Depending on whether you are cutting a minor/major or patch release, the process varies.

    * If you are cutting a new minor/major release:

        Create a new release branch (i.e release-X) and push it to the upstream repository.

        ```bash
            # Note: `upstream` must be the remote pointing to `github.com:kubernetes-sigs/cluster-api-operator`.
            git checkout -b release-0.14
            git push -u upstream release-0.14
            # Export the tag of the minor/major release to be cut, e.g.:
            export RELEASE_TAG=v0.14.0
        ```
    * If you are cutting a patch release from an existing release branch:

        Use existing release branch.

        ```bash
            # Note: `upstream` must be the remote pointing to `github.com:kubernetes-sigs/cluster-api-operator`
            git checkout upstream/release-0.14
            # Export the tag of the patch release to be cut, e.g.:
            export RELEASE_TAG=v0.14.1
        ```

3. Create a signed/annotated tag and push it:

```bash
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

This will trigger a [release GitHub action](https://github.com/kubernetes-sigs/cluster-api-operator/actions/workflows/release.yaml) that creates a release with operator components and the Helm chart. Concurrently, a Prow job will start to publish operator images to the staging registry.

4. Wait until images for the tag have been built and pushed to the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-capi-operator/global/cluster-api-operator) by the [post push images job](https://prow.k8s.io/?repo=kubernetes-sigs%2Fcluster-api-operator&job=post-cluster-api-operator-push-images).

5. If you don't have a GitHub token, create one by navigating to your GitHub settings, in [Personal access token](https://github.com/settings/tokens). Make sure you give the token the `repo` scope.

6. Create a PR to promote the images to the production registry:

```bash
# Export the tag of the release to be cut, e.g.:
export GITHUB_TOKEN=<your GH token>
export USER_FORK=<your GH account name>
make promote-images
```

**Notes**:
* `make promote-images` target tries to figure out your Github user handle in order to find the forked [k8s.io](https://github.com/kubernetes/k8s.io) repository.
    If you have not forked the repo, please do it before running the Makefile target.
* `kpromo` uses `git@github.com:...` as remote to push the branch for the PR. If you don't have `ssh` set up you can configure
    git to use `https` instead via `git config --global url."https://github.com/".insteadOf git@github.com:`.
* This will automatically create a PR in [k8s.io](https://github.com/kubernetes/k8s.io) and assign the CAPI Operator maintainers.


7. Merge the PR (/lgtm + /hold cancel) and verify the images are available in the production registry:
    * Wait for the [promotion prow job](https://prow.k8s.io/?repo=kubernetes%2Fk8s.io&job=post-k8sio-image-promo) to complete successfully. Then test the production image is accessible:

```bash
docker pull registry.k8s.io/capi-operator/cluster-api-operator:${RELEASE_TAG}
```

8. Publish the release in GitHub:

    * The draft release should be automatically created via the [release GitHub Action](https://github.com/kubernetes-sigs/cluster-api-operator/actions/workflows/release.yaml). Make sure that release is flagged as `pre-release` for all `beta` and `rc` releases or `latest` for a new release in the most recent release branch.

:tada: CONGRATULATIONS! The new [release](https://github.com/kubernetes-sigs/cluster-api-operator/releases) of CAPI Operator should be live now!!! :tada:

Please proceed to mandatory post release steps [next](#post-release-steps).

## Post-release steps

1. Switch back to the main branch and update `index.yaml` and `clusterctl-operator.yaml`. These are the sources for the operator Helm chart repository and the local krew plugin manifest index, respectively.

```bash
git checkout main
make update-helm-plugin-repo
```

2. Once run successfully, it will automatically create a PR against the operator repository with all the needed changes.

3. Depending on whether you are cutting a minor/major or patch release, next steps might be needed or redundant. Please follow along the next [chapter](#setup-jobs-and-dashboards-for-a-new-release-branch), in case this is a minor or major version release. 

## Setup jobs and dashboards for a new release branch
 
The goal of this task is to have test coverage for the new release branch and results in testgrid.
We are currently running CI jobs only in main and latest stable release branch (i.e release-0.14 is last minor release branch we created in earlier steps) and all configurations are hosted in test-infra [repository](https://github.com/kubernetes/test-infra). In this example, we will update `test-infra` repository jobs to track the new `release-0.14` branch.

1. Create new jobs based on the jobs running against our `main` branch:
    1. Rename `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-periodics-release-0-13.yaml` to `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-periodics-release-0-14.yaml`.
    2. Rename `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-presubmits-release-0-13.yaml` to `test-infra/config/jobs/kubernetes-sigs/cluster-api-operator/cluster-api-operator-presubmits-release-0-14.yaml`.
    3. Modify the following:
        1. Rename the jobs, e.g.: `periodic-cluster-api-operator-test-release-0-13` => `periodic-cluster-api-operator-test-release-0-14`.
        2. Change `annotations.testgrid-dashboards` to `sig-cluster-lifecycle-cluster-api-operator-0.14`.
        3. Change `annotations.testgrid-tab-name`, e.g. `capi-operator-test-release-0-13` => `capi-operator-test-release-0-14`.
        4. For periodics additionally:
            * Change `extra_refs[].base_ref` to `release-0.14` (for repo: `cluster-api-operator`).
        5. For presubmits additionally: Adjust branches: `^release-0.13$` => `^release-0.14$`.
2. Create a new dashboard for the new branch in: `test-infra/config/testgrids/kubernetes/sig-cluster-lifecycle/config.yaml` (`dashboard_groups` and `dashboards`).
    * Modify a previous job entry: `sig-cluster-lifecycle-cluster-api-operator-0.13` => `sig-cluster-lifecycle-cluster-api-operator-0.14` in both `dashboard_groups` and `dashboards` lists.
3. Verify the jobs and dashboards a day later by taking a look at: `https://testgrid.k8s.io/sig-cluster-lifecycle-cluster-api-operator-0.14`.

Prior art:
- https://github.com/kubernetes/test-infra/pull/30372
- https://github.com/kubernetes/test-infra/pull/33506