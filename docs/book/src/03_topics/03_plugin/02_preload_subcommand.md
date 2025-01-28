# Using the `preload` Plugin for Kubernetes Operator

## Overview

The `preload` subcommand allows users to preload provider `ConfigMaps` into a management cluster from an OCI (Open Container Initiative) artifact, known provider source, or URL override. Users can supply any number of provider stings or discover and use existing provider manifests from the cluster.

## Command Syntax
The basic syntax for using the `preload` command is:

```sh
kubectl operator preload [flags]
```

## Flags and Options
| Flag | Short | Description |
|------|-------|-------------|
| `--kubeconfig` | | Path to the kubeconfig file for the source management cluster. Uses default discovery rules if unspecified. |
| `--existing` | `-e` | Discover all providers in the cluster and prepare `ConfigMap` for each of them. |
| `--core` | | Specifies the core provider and version (e.g., `cluster-api:v1.1.5`). Defaults to the latest release. |
| `--infrastructure` | `-i` | Specifies infrastructure providers and versions (e.g., `aws:v0.5.0`). |
| `--bootstrap` | `-b` | Specifies bootstrap providers and versions (e.g., `kubeadm:v1.1.5`). |
| `--control-plane` | `-c` | Specifies control plane providers and versions (e.g., `kubeadm:v1.1.5`). |
| `--ipam` | | Specifies IPAM providers and versions (e.g., `infoblox:v0.0.1`). |
| `--runtime-extension` | | Specifies runtime extension providers and versions (e.g., `my-extension:v0.0.1`). |
| `--addon` | | Specifies add-on providers and versions (e.g., `helm:v0.1.0`). |
| `--target-namespace` | `-n` | Specifies the target namespace where the operator should be deployed. Defaults to `capi-operator-system`. |
| `--artifact-url` | `-u` | Specifies the URL of the OCI artifact containing component manifests. |

## Examples

### Load CAPI Operator Manifests from an OCI Source
```sh
kubectl operator preload --core cluster-api
```
This command loads the `cluster-api` core provider manifests into the management cluster. If no version is specified, the latest release is used.

### Load CAPI Operator Manifests from Existing Providers in the Cluster
```sh
kubectl operator preload -e
```
This command discovers all existing providers in the cluster and prepares ConfigMaps containing their manifests.

### Prepare Provider ConfigMap from OCI for a Specific Infrastructure Provider
```sh
kubectl operator preload --infrastructure=aws -u my-registry.example.com/infrastructure-provider
```
This command fetches the latest available version of the `aws` infrastructure provider from the specified OCI registry and creates a ConfigMap.

### Prepare Provider ConfigMap with a Specific Version
```sh
kubectl operator preload --infrastructure=aws::v2.3.0 -u my-registry.example.com/infrastructure-provider
```
This command loads the AWS infrastructure provider version `v2.3.0` from the OCI registry into the default namespace.

### Prepare Provider ConfigMap with a Custom Namespace
```sh
kubectl operator preload --infrastructure=aws:custom-namespace -u my-registry.example.com/infrastructure-provider
```
This command loads the latest version of the AWS infrastructure provider into the `custom-namespace`.

### Prepare Provider ConfigMap with a Specific Version and Namespace
```sh
kubectl operator preload --infrastructure=aws:custom-namespace:v2.3.0 -u my-registry.example.com/infrastructure-provider
```
This command loads AWS provider version `v2.3.0` into `custom-namespace`.

### Prepare Provider ConfigMap for Multiple Infrastructure Providers
```sh
kubectl operator preload --infrastructure=aws --infrastructure=vsphere -u my-registry.example.com/infrastructure-provider
```
This command fetches and loads manifests for both AWS and vSphere infrastructure providers from the OCI registry.

### Prepare Provider ConfigMap with a Custom Target Namespace
```sh
kubectl operator preload --infrastructure aws --target-namespace foo -u my-registry.example.com/infrastructure-provider
```
This command loads the AWS infrastructure provider into the `foo` namespace, ensuring that the operator uses a customized deployment location.
