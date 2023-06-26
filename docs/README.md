# Table of Contents

- [Introduction](#introduction)
  * [Overview](#overview)
  * [Features](#features)
- [Getting started](#getting-started)
  * [Glossary](#glossary)
  * [Prerequisites](#prerequisites)
  * [Installation](#installation)
    + [Method 1: Apply Manifests from Release Assets](#method-1-apply-manifests-from-release-assets)
    + [Method 2: Use Helm Charts](#method-2-use-helm-charts)
  * [Configuration](#configuration)
    + [Examples of Configuration Options](#examples-of-configuration-options)
  * [Basic Cluster API Provider Installation](#basic-cluster-api-provider-installation)
    + [Installing the CoreProvider](#installing-the-coreprovider)
    + [Installing Azure Infrastructure Provider](#installing-azure-infrastructure-provider)
    + [Deleting providers](#deleting-providers)
- [Custom Resource Definitions (CRDs)](#custom-resource-definitions-crds)
  * [Overview](#overview-1)
  * [Provider Spec](#provider-spec)
  * [Provider Status](#provider-status)
- [Examples of API Usage](#examples-of-api-usage)
- [Cluster API Provider Lifecycle](#cluster-api-provider-lifecycle)
  * [Installing a Provider](#installing-a-provider)
  * [Upgrading a Provider](#upgrading-a-provider)
  * [Modifying a Provider](#modifying-a-provider)
  * [Deleting a Provider](#deleting-a-provider)
- [Air-gapped Environment](#air-gapped-environment)

# Introduction

## Overview

The **Cluster API Operator** is a Kubernetes Operator designed to empower cluster administrators to handle the lifecycle of Cluster API providers within a management cluster using a declarative approach. It aims to improve user experience in deploying and managing Cluster API, making it easier to handle day-to-day tasks and automate workflows with GitOps. 

This operator leverages a declarative API and extends the capabilities of the `clusterctl` CLI, allowing greater flexibility and configuration options for cluster administrators.

## Features

- Offers a **declarative API** that simplifies the management of Cluster API providers and enables GitOps workflows.
- Facilitates **provider upgrades and downgrades** making it more convenient for distributed teams and CI pipelines.
- Aims to support **air-gapped environments** without direct access to GitHub/GitLab.
- Leverages **controller-runtime** configuration API for a more flexible Cluster API providers setup.
- Provides a **transparent and effective** way to interact with various Cluster API components on the management cluster.

# Getting started

## Glossary

The lexicon used in this document is described in more detail [here](https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/book/src/reference/glossary.md). Any discrepancies should be rectified in the main Cluster API glossary.

## Prerequisites

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for interacting with the management cluster.
- [Helm](https://helm.sh/docs/intro/install/) for installing operator on the cluster (optional).

## Installation

Before installing the Cluster API Operator, you must first ensure that cert-manager is installed, as the operator does not manage cert-manager installations. To install cert-manager, run the following command:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for cert-manager to be ready before proceeding.

After cert-manager is successfully installed, you can install the Cluster API operator using one of the following methods:

### Method 1: Apply Manifests from Release Assets

Install the Cluster API operator directly by applying the latest release assets:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/cluster-api-operator/releases/latest/download/operator-components.yaml
```

### Method 2: Use Helm Charts

Alternatively, you can install the Cluster API operator using Helm charts:

```bash
helm repo add capi-operator https://kubernetes-sigs.github.io/cluster-api-operator
helm repo update
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system
```

⚠️ **Note:** Make sure to review and adjust the RBAC permissions as needed. The operator will create and update CRDs, so appropriate permissions should be granted. We are continuously working to determine the best way to handle this.

## Configuration

The Cluster API Operator uses the controller-runtime library, making it compatible with all the options that the library provides. This offers flexibility when configuring the operator and allows you to benefit from the features offered by controller-runtime.

### Examples of Configuration Options

Some examples of controller-runtime configuration options you can use with the Cluster API Operator include:

1. **Metrics:** Controller-runtime enables you to collect and expose metrics about its internal behavior, such as the number of reconciliations executed by the operator over time. You can customize the metrics endpoint and the metrics scraping interval, among other settings.

2. **Leader Election:** To ensure high availability of the operator, you can enable leader election when running multiple replicas. Controller-runtime allows you to set the leader election resource lock and polling interval to suit your needs.

3. **Logger:** The operator allows you to use controller-runtime logging options to configure the logging subsystem. You can choose the logging level and output format, and even enable logging for specific libraries or components.

Here's an example of how you can configure the Cluster API Operator deployment with some of these options:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-api-operator
  namespace: capi-operator-system
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-bind-addr=:8080
        - --leader-elect
        - --leader-elect-retry-period=5s
        - --v=5
        env:...
```

For complete details on the available configuration options, you can execute:

```bash
export CAPI_OPERATOR_VERSION=v0.3.0
docker run -it --rm registry.k8s.io/capi-operator/cluster-api-operator:${CAPI_OPERATOR_VERSION} /manager --help
```

## Basic Cluster API Provider Installation

In this section, we will walk you through the basic process of installing Cluster API providers using the operator. The Cluster API operator manages four types of objects:

- CoreProvider
- BootstrapProvider
- ControlPlaneProvider
- InfrastructureProvider

Please note that this example provides a basic configuration of Azure Infrastructure provider for getting started. More detailed examples and CRD descriptions will be provided in subsequent sections of this document.

### Installing the CoreProvider

The first step is to install the CoreProvider, which is responsible for managing the Cluster API CRDs and the Cluster API controller.

You can utilize any existing namespace for providers in your Kubernetes operator. However, before creating a provider object, make sure the specified namespace has been created. In the example below, we use the `capi-system` namespace. You can create this namespace through either the Command Line Interface (CLI) by running `kubectl create namespace capi-system`, or by using the declarative approach described in the [official Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/namespaces-walkthrough/#create-new-namespaces).

*Example:*

```yaml
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  version: v1.4.3
```

**Note:** Only one CoreProvider can be installed at the same time on a single cluster.

### Installing Azure Infrastructure Provider

Next, install [Azure Infrastructure Provider](https://capz.sigs.k8s.io/). Before that ensure that `capz-system` namespace exists.

Since the provider requires variables to be set, create a secret containing them in the same namespace as the provider. It is also recommended to include a `github-token` in the secret. This token is used to fetch the provider repository, and it is required for the provider to be installed. The operator may exceed the rate limit of the GitHub API without the token. Like [clusterctl](https://cluster-api.sigs.k8s.io/clusterctl/overview.html?highlight=github_token#avoiding-github-rate-limiting), the token needs only the `repo` scope.

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
 version: v1.9.3
 secretName: azure-variables
```

### Deleting providers

To remove the installed providers and all related kubernetes objects just delete the following CRs:

```bash
kubectl delete coreprovider cluster-api
kubectl delete infrastructureprovider azure
```

# Custom Resource Definitions (CRDs)

## Overview

The Cluster API Operator introduces new API types: `CoreProvider`, `BootstrapProvider`, `ControlPlaneProvider`, and `InfrastructureProvider`. These four provider types share common Spec and Status types, `ProviderSpec` and `ProviderStatus`, respectively.

The CRDs are scoped to be namespaced, allowing RBAC restrictions to be enforced if needed. This scoping also enables the installation of multiple versions of controllers (grouped within namespaces) in the same management cluster. 

To better understand how the API can be used, please refer to the [Example API Usage section](#example-api-usage).

Related Golang structs can be found in the [Cluster API Operator repository](https://github.com/kubernetes-sigs/cluster-api-operator/tree/main/api/v1alpha1).

Below are the new API types being defined, with shared types used for Spec and Status among the different provider types—Core, Bootstrap, ControlPlane, and Infrastructure:

*CoreProvider*
```golang
type CoreProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*BootstrapProvider*
```golang
type BootstrapProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*ControlPlaneProvider*
```golang
type ControlPlaneProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*InfrastructureProvider*
```golang
type InfrastructureProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

The following sections provide details about `ProviderSpec` and `ProviderStatus`, which are shared among all the provider types: Core, Bootstrap, ControlPlane, and Infrastructure.

## Provider Spec

1. `ProviderSpec`: desired state of the Provider, consisting of:
   - Version (string): provider version (e.g., "v0.1.0")
   - Manager (optional ManagerSpec): controller manager properties for the provider
   - Deployment (optional DeploymentSpec): deployment properties for the provider
   - SecretName (optional string): name of the secret that contains provider credentials
   - SecretNamespace (optional string): namespace of the secret that contains provider credentials
   - FetchConfig (optional FetchConfiguration): how the operator will fetch components and metadata

   YAML example:
   ```yaml
   ...
   spec:
    version: "v0.1.0"
    manager:
      maxConcurrentReconciles: 5
    deployment:
      replicas: 1
    secretName: "provider-secret"
    fetchConfig:
      url: "https://github.com/owner/repo/releases"
   ...
   ```

2. `ManagerSpec`: controller manager properties for the provider, consisting of:
   - ProfilerAddress (optional string): pprof profiler bind address (e.g., "localhost:6060")
   - MaxConcurrentReconciles (optional int): maximum number of concurrent reconciles
   - Verbosity (optional int): logs verbosity
   - FeatureGates (optional map[string]bool): provider specific feature flags

   YAML example:
   ```yaml
   ...
   spec:
    manager:
      profilerAddress: "localhost:6060"
      maxConcurrentReconciles: 5
      verbosity: 1
      featureGates:
        FeatureA: true
        FeatureB: false
   ...
   ```

3. `DeploymentSpec`: deployment properties for the provider, consisting of:
   - Replicas (optional int): number of desired pods
   - NodeSelector (optional map[string]string): node label selector
   - Tolerations (optional []corev1.Toleration): pod tolerations
   - Affinity (optional corev1.Affinity): pod scheduling constraints
   - Containers (optional []ContainerSpec): list of deployment containers
   - ServiceAccountName (optional string): pod service account
   - ImagePullSecrets (optional []corev1.LocalObjectReference): list of image pull secrets specified in the Deployment

   YAML example:
   ```yaml
   ...
   spec:
     deployment:
       replicas: 2
       nodeSelector:
         disktype: ssd
       tolerations:
       - key: "example"
         operator: "Exists"
         effect: "NoSchedule"
       affinity:
         nodeAffinity:
           requiredDuringSchedulingIgnoredDuringExecution:
             nodeSelectorTerms:
             - matchExpressions:
               - key: "example"
                 operator: "In"
                 values:
                 - "true"
       containers:
         - name: "containerA"
           image:
             repository: "example.com/repo"
             name: "image-name"
             tag: "v1.0.0"
           args:
             exampleArg: "value"
    ...
   ```

4. `ContainerSpec`: container properties for the provider, consisting of:
   - Name (string): container name
   - Image (optional ImageMeta): container image metadata
   - Args (optional map[string]string): extra provider specific flags
   - Env (optional []corev1.EnvVar): environment variables
   - Resources (optional corev1.ResourceRequirements): compute resources
   - Command (optional []string): override container's entrypoint array

   YAML example:
   ```yaml
   ...
   spec:
     deployment:
       containers:
         - name: "example-container"
           image:
             repository: "example.com/repo"
             name: "image-name"
             tag: "v1.0.0"
           args:
             exampleArg: "value"
           env:
             - name: "EXAMPLE_ENV"
               value: "example-value"
           resources:
             limits:
               cpu: "1"
               memory: "1Gi"
             requests:
               cpu: "500m"
               memory: "500Mi"
           command:
             - "/bin/bash"
   ...
   ```

5. `ImageMeta`: container image customization, consisting of:
   - Repository (optional string): image registry (e.g., "example.com/repo")
   - Name (optional string): image name (e.g., "provider-image")
   - Tag (optional string): image tag (e.g., "v1.0.0")

6. `FetchConfiguration`: components and metadata fetch options, consisting of:
   - URL (optional string): URL for remote Github repository releases (e.g., "https://github.com/owner/repo/releases")
   - Selector (optional metav1.LabelSelector): label selector to use for fetching provider components and metadata from ConfigMaps stored in the cluster

   YAML example:
   ```yaml
   ...
   spec:
     fetchConfig:
       url: "https://github.com/owner/repo/releases"
       selector:
         matchLabels:
   ...
   ```
## Provider Status

`ProviderStatus`: observed state of the Provider, consisting of:
   - Contract (optional string): core provider contract being adhered to (e.g., "v1beta1")
   - Conditions (optional clusterv1.Conditions): current service state of the provider
   - ObservedGeneration (optional int64): latest generation observed by the controller
   - InstalledVersion (optional string): version of the provider that is installed

   YAML example:
   ```yaml
   status:
     contract: "v1beta1"
     conditions:
       - type: "Ready"
         status: "True"
         reason: "ProviderAvailable"
         message: "Provider is available and ready"
     observedGeneration: 1
     installedVersion: "v0.1.0"
   ```

# Examples of API Usage

In this section we provide some concrete examples of CAPI Operator API usage for various use-cases.

1. As an admin, I want to install the aws infrastructure provider with specific controller flags.

```yaml
apiVersion: v1
kind: Secret
metadata:
 name: aws-variables
 namespace: capa-system
type: Opaque
data:
 AWS_B64ENCODED_CREDENTIALS: ...
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: aws
 namespace: capa-system
spec:
 version: v2.1.4
 secretName: aws-variables
 manager:
   # These top level controller manager flags, supported by all the providers.
   # These flags come with sensible defaults, thus requiring no or minimal
   # changes for the most common scenarios.
   metrics:
    bindAddress: ":8181"
   syncPeriod: "500s"
 fetchConfig:
   url: https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases
 deployment:
   containers:
   - name: manager
     args:
      # These are controller flags that are specific to a provider; usage
      # is reserved for advanced scenarios only.
      "--awscluster-concurrency": "12"
      "--awsmachine-concurrency": "11"
```

2. As an admin, I want to install aws infrastructure provider but override the container image of the CAPA deployment.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: aws
 namespace: capa-system
spec:
 version: v2.1.4
 secretName: aws-variables
 deployment:
   containers:
   - name: manager
     image:
       repository: "gcr.io/myregistry"
       name: "capa-controller"
       tag: "v2.1.4-foo"
```

3. As an admin, I want to change the resource limits for the manager pod in my control plane provider deployment.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: ControlPlaneProvider
metadata:
 name: kubeadm
 namespace: capi-kubeadm-control-plane-system
spec:
 version: v1.4.3
 secretName: capi-variables
 deployment:
   containers:
   - name: manager
     resources:
       limits:
         cpu: 100m
         memory: 30Mi
       requests:
         cpu: 100m
         memory: 20Mi
```

4. As an admin, I would like to fetch my azure provider components from a specific repository which is not the default.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: myazure
 namespace: capz-system
spec:
 version: v1.9.3
 secretName: azure-variables
 fetchConfig:
   url: https://github.com/myorg/awesome-azure-provider/releases

```

5. As an admin, I would like to use the default fetch configurations by simply specifying the expected Cluster API provider names such as `aws`, `vsphere`, `azure`, `kubeadm`, `talos`, or `cluster-api` instead of having to explicitly specify the fetch configuration. In the example below, since we are using 'vsphere' as the name of the InfrastructureProvider the operator will fetch it's configuration from `url: https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/releases` by default.

See more examples in the [air-gapped environment section](#air-gapped-environment)

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: vsphere
 namespace: capv-system
spec:
 version: v1.6.1
 secretName: vsphere-variables
```

# Cluster API Provider Lifecycle

This Section covers the lifecycle of Cluster API providers managed by the Cluster API Operator, including installing, upgrading, modifying, and deleting a provider.

## Installing a Provider

To install a new Cluster API provider with the Cluster API Operator, create a provider object as shown in the first example API usage for creating the secret with variables and the provider itself.

The operator processes a provider object by applying the following rules:

- The CoreProvider is installed first; other providers will be requeued until the core provider exists.
- Before installing any provider, the following pre-flight checks are executed:
    - No other instance of the same provider (same Kind, same name) should exist in any namespace.
    - The Cluster API contract (e.g., v1beta1) must match the contract of the core provider.
- The operator sets conditions on the provider object to surface any installation issues, including pre-flight checks and/or order of installation.
- If the FetchConfiguration is not defined, the operator applies the embedded fetch configuration for the given kind and `ObjectMeta.Name` specified in the [Cluster API code](https://github.com/kubernetes-sigs/cluster-api/blob/main/cmd/clusterctl/client/config/providers_client.go).

The installation process, managed by the operator, aligns with the implementation underlying the `clusterctl init` command and includes these steps:

- Fetching provider artifacts (the components.yaml and metadata.yaml files).
- Applying image overrides, if any.
- Replacing variables in the infrastructure-components from EnvVar and Secret.
- Applying the resulting YAML to the cluster.

Differences between the operator and `clusterctl init` include:

- The operator installs one provider at a time while `clusterctl init` installs a group of providers in a single operation.
- The operator stores fetched artifacts in a config map for reuse during subsequent reconciliations.
- The operator uses a Secret, while `clusterctl init` relies on environment variables and a local configuration file.

## Upgrading a Provider

To trigger an upgrade for a Cluster API provider, change the `spec.Version` field. All providers must follow the golden rule of respecting the same Cluster API contract supported by the core provider.

The operator performs the upgrade by:

1. Deleting the current provider components, while preserving CRDs, namespaces, and user objects.
2. Installing the new provider components.

Differences between the operator and `clusterctl upgrade apply` include:

- The operator upgrades one provider at a time while `clusterctl upgrade apply` upgrades a group of providers in a single operation.
- With the declarative approach, users are responsible for manually editing the Provider objects' YAML, while `clusterctl upgrade apply --contract` automatically determines the latest available versions for each provider.

## Modifying a Provider

In addition to changing a provider version (upgrades), the operator supports modifying other provider fields such as controller flags and variables. This can be achieved through `kubectl edit` or `kubectl apply` to the provider object.

The operation works similarly to upgrades: The current provider instance is deleted while preserving CRDs, namespaces, and user objects. Then, a new provider instance with the updated flags/variables is installed.

**Note**: `clusterctl` currently does not support this operation.

## Deleting a Provider

To delete a provider, remove the corresponding provider object. Provider deletion will be blocked if any workload clusters using the provider still exist. Furthermore, deletion of a core provider is blocked if other providers remain in the management cluster.

## Air-gapped Environment

To install Cluster API providers in an air-gapped environment using the operator, address the following issues:

1. Configure the operator for an air-gapped environment:
   - Manually fetch and store a helm chart for the operator.
   - Provide image overrides for the operator in from an accessible image repository.
2. Configure providers for an air-gapped environment:
   - Provide fetch configuration for each provider from an accessible location (e.g., an internal GitHub repository) or from pre-created ConfigMaps within the cluster.
   - Provide image overrides for each provider to pull images from an accessible image repository.

**Example Usage:**

As an admin, I need to fetch the Azure provider components from within the cluster because I am working in an air-gapped environment.

In this example, there is a ConfigMap in the `capz-system` namespace that defines the components and metadata of the provider.

The Azure InfrastructureProvider is configured with a `fetchConfig` specifying the label selector, allowing the operator to determine the available versions of the Azure provider. Since the provider's version is marked as `v1.9.3`, the operator uses the components information from the ConfigMap with matching label to install the Azure provider.

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    provider-components: azure
  name: v1.9.3
  namespace: capz-system
data:
  components: |
    # Components for v1.9.3 YAML go here
  metadata: |
    # Metadata information goes here
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
  name: azure
  namespace: capz-system
spec:
  version: v1.9.3
  secretName: azure-variables
  fetchConfig:
    selector:
      matchLabels:
        provider-components: azure
