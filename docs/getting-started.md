# Getting started

This document contains instructions on how to start with Cluster API operator.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for interacting with the cluster.
- [Kind](https://kind.sigs.k8s.io/#installation-and-usage) for creating a local cluster.

## Installation

Create a cluster using kind:

```bash
kind create cluster
```

Cluster API Operator doesn't manage cert-manager installations, you have to install it manually:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for the cert-manager to be ready.

Install the Cluster API operator:

1. Operator can be installed directly by applying manifests from release assets:
```bash
kubectl apply -f https://github.com/kubernetes-sigs/cluster-api-operator/releases/latest/download/operator-components.yaml
```
2. Another option is using helm charts:
```bash
helm repo add capi-operator https://kubernetes-sigs.github.io/cluster-api-operator
helm repo update
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system
```

***Note***: :warning: Take a look at RBAC permissions and adjust them, the operator will be creating and updating CRDs.
We are still working on figuring out the best way to handle this.

## Usage

There are 4 types of objects that are managed by the Cluster API operator:

- CoreProvider
- BootstrapProvider
- ControlPlaneProvider
- InfrastructureProvider

First, CoreProvider has to be installed. CoreProvider is responsible for managing the Cluster API CRDs and the Cluster API controller.

Example:
```yaml
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  version: v1.3.2
```

**Note**: Only one CoreProvider can be installed at the same time on one cluster. Any namespace can be used for the CoreProvider.

Next, BootstrapProvider, ControlPlaneProvider and InfrastructureProvider can be installed. They are responsible for managing the CRDs and the controllers for the corresponding provider.

If provider requires variables to be set, a secret containing them has to be created and it has to be in the same namespace as the provider.

It's also recommended to include github-token in the secret. This token is used to fetch the provider repository and it is required for the provider to be installed. 
Operator might exceed the rate limit of the github API without the token.

Example:
```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: azure-variables
  namespace: capz-system
type: Opaque
stringData:
  AZURE_CLIENT_ID_B64: Zm9vCg==
  AZURE_CLIENT_SECRET_B64: Zm9vCg==
  AZURE_SUBSCRIPTION_ID_B64: Zm9vCg==
  AZURE_TENANT_ID_B64: Zm9vCg==
  github-token: ghp_fff
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: azure
 namespace: capz-system
spec:
 version: v1.7.2
 secretName: azure-variables
```
## Upgrading providers

To upgrade a provider, modify the `spec.Version` field of the provider object.

## Air gapped environment

In order to install Cluster API providers in an air-gapped environment the following steps are supported:

- If you need to provide image overrides for any provider modify `provider.Spec.Deployment.Containers[].Image`.
- For reading provider components from an accessible location (e.g. an internal github repository) modify `provider.Spec.FetchConfig.Url`, or `provider.Spec.FetchConfig.Selector` for using a ConfigMap. The ConfigMap is expected to contain components and metadata for a specific version only.
The name of the ConfigMap should be set to the provider version or to override this add a label like the following: `provider.cluster.x-k8s.io/version=v1.4.3`

