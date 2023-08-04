# Releasing new versions

This documents describes release process for the Cluster API Operator.

1. Create the release branch and cut release tag.

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

This will trigger [release github action](https://github.com/kubernetes-sigs/cluster-api-operator/blob/main/.github/workflows/release.yaml) that will
create a draft release with operator components and helm chart, also a Prow job to publish operator image to the staging registry will start.

2. Wait for image to appear in the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-capi-operator/global/cluster-api-operator).

3. Create a GitHub [Personal access tokens](https://github.com/settings/tokens) if you don't have one already. We're going to use for opening a PR
to promote image from staging to production.

```bash
export GITHUB_TOKEN=<your GH token>
make promote-images
```

Merge the PR after it was created and verify that image is present in the production registry.

```bash
docker pull registry.k8s.io/capi-operator/cluster-api-operator:${RELEASE_TAG}
```

4. Publish the release on Github.

5. After release was published, switch back to main branch and update index.yaml. It's the source for operator helm chart repository.

```bash
git checkout main
make update-helm-repo
```

6. Create a PR with the changes.
