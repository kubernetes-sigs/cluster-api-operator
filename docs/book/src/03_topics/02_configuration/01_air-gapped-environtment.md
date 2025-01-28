# Air-gapped Environment

To install Cluster API providers in an air-gapped environment using the operator, address the following issues:

1. Configure the operator for an air-gapped environment:
   - Manually fetch and store a helm chart for the operator.
   - Provide image overrides for the operator from an accessible image repository.
2. Configure providers for an air-gapped environment:
   - Provide fetch configuration for each provider from an accessible location: e.g., an OCI artifact, internal Github/Gitlab repository URL or from pre-created ConfigMaps within the cluster.
   - Provide image overrides for each provider to pull images from an accessible image repository.

**Example Usage:**

As an admin, I need to fetch the Azure provider components from within the cluster because I am working in an air-gapped environment.

### Using ConfigMap

In this example, there is a ConfigMap in the `capz-system` namespace that defines the components and metadata of the provider.

The Azure InfrastructureProvider is configured with a `fetchConfig` specifying the label selector, allowing the operator to determine the available versions of the Azure provider. Since the provider's version is marked as `v1.9.3`, the operator uses the components information from the ConfigMap with a matching label to install the Azure provider.

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

### Using OCI Artifact

OCI artifact files can follow these naming patterns:

- `<registry>/<repository>:<tag>` (e.g., `my-registry.example.com/my-provider:v1.9.3`)
- `<registry>/<repository>` (e.g., my-registry.example.com/my-provider), in which case the tag is substituted by provider version.

When working with metadata and component files within OCI artifacts, the files stored in the artifact should follow these naming conventions:

- **Metadata Files**:
  - Default: `metadata.yaml`
  - Versioned: `fmt.Sprintf("%s-%s-%s-metadata.yaml", p.GetType(), p.GetName(), p.GetSpec().Version)`, Example: `infrastructure-azure-v1.9.3-metadata.yaml`

- **Component Files**:
  - Default: `components.yaml`
  - Typed: `fmt.Sprintf("%s-components.yaml", p.GetType())`, Example: `infrastructure-components.yaml`
  - Versioned: `fmt.Sprintf("%s-%s-%s-components.yaml", p.GetType(), p.GetName(), p.GetSpec().Version)`, Example: `infrastructure-azure-v1.9.3-components.yaml`

Versioned files allow to use single image for hosting multiple provider manifests and versions simultaneously, without overlapping each other.

Typed allow to store multiple provider types inside single image, which is needed for example for `bootstrap` and `control-plane` providers.

Example layout for a `kubeadm` provider may look like:
- `metadata.yaml`
- `control-plane-components.yaml`
- `bootstrap-components.yaml`

To fetch provider components which are stored as an OCI artifact, you can configure `fetchConfig.oci` field to pull them directly from an OCI registry:

```yaml
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
    oci: "my-oci-registry.example.com/my-provider:v1.9.3"
```

## OCI Authentication

To securely authenticate with an OCI registry, environment variables are used for user credentials. The following environment variables are involved:

- **`OCI_USERNAME`**: The username for the OCI registry.
- **`OCI_PASSWORD`**: The password associated with the username.
- **`OCI_ACCESS_TOKEN`**: A token used for authentication.
- **`OCI_REFRESH_TOKEN`**: A refresh token to obtain new access tokens.

### Fetching Provider Components from a secure OCI Registry

To fetch provider components stored as an OCI artifact, you can configure the `fetchConfig.oci` field to pull them directly from an OCI registry. The `configSecret` field references a Kubernetes `Secret` that should contain the necessary OCI credentials (such as username and password, or token), ensuring that sensitive information is securely stored.

Here’s an example of how to configure the `InfrastructureProvider` resource to fetch a specific version of a provider component from an OCI registry:

```yaml
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
  name: azure
  namespace: capz-system
spec:
  version: v1.9.3
  configSecret:
    name: azure-variables  # Secret containing the OCI registry credentials
  fetchConfig:
    oci: "my-oci-registry.example.com/my-provider:v1.9.3"  # Reference to the OCI artifact (provider)
```

The reference secret can could contain OCI authentication data:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-variables  # Name of the secret referenced in the InfrastructureProvider
  namespace: capz-system  # Namespace where the secret resides
type: Opaque
data:
  OCI_USERNAME: <secret>
  OCI_PASSWORD: <secret>
  OCI_ACCESS_TOKEN: <secret>
  OCI_REFRESH_TOKEN: <secret>
```

### Using Github/Gitlab URL

If the provider components are hosted at a specific repository URL, you can use `fetchConfig.url` to retrieve them directly.

```yaml
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
    url: "https://my-internal-repo.example.com/providers/azure/v1.9.3.yaml"
```

## Situation when manifests do not fit into ConfigMap

There is a limit on the [maximum size](https://kubernetes.io/docs/concepts/configuration/configmap/#motivation) of a ConfigMap - 1MiB. If the manifests do not fit into this size, Kubernetes will generate an error and provider installation will fail. To avoid this, you can archive the manifests and put them in the ConfigMap that way.

For example, you have two files: `components.yaml` and `metadata.yaml`. To create a working ConfigMap, you need:

1. Archive components.yaml using `gzip` CLI tool:

```sh
gzip -c components.yaml > components.gz
```

2. Create a ConfigMap in your Kubernetes cluster from the archived data:

```sh
kubectl create configmap v1.9.3 -n capz-system --from-file=components=components.gz --from-file=metadata=metadata.yaml
```

3. Add "provider.cluster.x-k8s.io/compressed: true" annotation to the ConfigMap:

```sh
kubectl annotate configmap v1.9.3 -n capz-system provider.cluster.x-k8s.io/compressed=true
```

**Note**: Without this annotation, the operator won't be able to determine if the data is compressed or not.

4. Add labels that will be used to match the ConfigMap in the `fetchConfig` section of the provider:

```sh
kubectl label configmap v1.9.3 -n capz-system provider-components=azure
```
