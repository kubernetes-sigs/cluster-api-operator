# Air-gapped Environment

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
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
  name: azure
  namespace: capz-system
spec:
  version: v1.9.3
  configSecret:
    name: azure-variables
  fetchConfig:
    selector:
      matchLabels:
        provider-components: azure
```

### Situation when manifests do not fit into configmap

There is a limit on the [maximum size](https://kubernetes.io/docs/concepts/configuration/configmap/#motivation) of a configmap - 1MiB. If the manifests do not fit into this size, Kubernetes will generate an error and provider installation fail. To avoid this, you can archive the manifests and put them in the configmap that way.

For example, you have two files: `components.yaml` and `metadata.yaml`. To create a working config map you need:

1. Archive components.yaml using `gzip` cli tool

```sh
gzip -c components.yaml > components.gz
```

2. Create a configmap manifest from the archived data

```sh
kubectl create configmap v1.9.3 --namespace=capz-system --from-file=components=components.gz --from-file=metadata=metadata.yaml --dry-run=client -o yaml > configmap.yaml
```

3. Edit the file by adding "provider.cluster.x-k8s.io/compressed: true" annotation

```sh
yq eval -i '.metadata.annotations += {"provider.cluster.x-k8s.io/compressed": "true"}' configmap.yaml
```

**Note**: without this annotation operator won't be able to determine if the data is compressed or not.

4. Add labels that will be used to match the configmap in `fetchConfig` section of the provider

```sh
yq eval -i '.metadata.labels += {"my-label": "label-value"}' configmap.yaml
```

5. Create a configmap in your kubernetes cluster using kubectl

```sh
kubectl create -f configmap.yaml
```
