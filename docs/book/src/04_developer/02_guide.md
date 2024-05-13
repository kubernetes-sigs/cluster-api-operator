# Developer Guide

## Prerequisites

### Docker

Iterating on the Cluster API Operator involves repeatedly building Docker containers.

[docker]: https://docs.docker.com/install/

### A Cluster

You'll likely want an existing cluster as your [management cluster][mcluster].
The easiest way to do this is with [kind] v0.9 or newer, as explained in the quick start.

Make sure your cluster is set as the default for `kubectl`.
If it's not, you will need to modify subsequent `kubectl` commands below.

[mcluster]: ../reference/glossary.md#management-cluster
[kind]: https://github.com/kubernetes-sigs/kind

### kubectl

[kubectl] for interacting with the management cluster.

[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/

### Helm

[Helm] for installing operator on the cluster (optional).

[Helm]: https://helm.sh/docs/intro/install/

### A container registry

If you're using [kind], you'll need a way to push your images to a registry so they can be pulled.
You can instead [side-load] all images, but the registry workflow is lower-friction.

Most users test with [GCR], but you could also use something like [Docker Hub][hub].
If you choose not to use GCR, you'll need to set the `REGISTRY` environment variable.

[side-load]: https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster
[GCR]: https://cloud.google.com/container-registry/
[hub]: https://hub.docker.com/

### Kustomize

You'll need to [install `kustomize`][kustomize].
There is a version of `kustomize` built into kubectl, but it does not have all the features of `kustomize` v3 and will not work.

[kustomize]: https://kubectl.docs.kubernetes.io/installation/kustomize/

### Kubebuilder

You'll need to [install `kubebuilder`][kubebuilder].

[kubebuilder]: https://book.kubebuilder.io/quick-start.html#installation

### Cert-Manager

You'll need to deploy [cert-manager] components on your [management cluster][mcluster], using `kubectl`

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.2/cert-manager.yaml
```

Ensure the cert-manager webhook service is ready before creating the Cluster API Operator components.

This can be done by following instructions for [manual verification](https://cert-manager.io/docs/installation/verify/#manual-verification)
from the [cert-manager] website.
Note: make sure to follow instructions for the release of cert-manager you are installing.

[cert-manager]: https://github.com/cert-manager/cert-manager

## Development

## Option 1: Tilt

[Tilt][tilt] is a tool for quickly building, pushing, and reloading Docker containers as part of a Kubernetes deployment.

Once you have a running Kubernetes cluster, you can run:

```bash
tilt up
```

That's it! Tilt will automatically reload the deployment to your local cluster every time you make a code change.

[tilt]: https://tilt.dev

## Option 2: The kustomize way

```bash
# Build all the images
make docker-build

# Push images
make docker-push

# Apply the manifests
kustomize build config/default | ./hack/tools/bin/envsubst | kubectl apply -f -
```


