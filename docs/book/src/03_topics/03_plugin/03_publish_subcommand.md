# Using the `publish` Subcommand

The `publish` subcommand allows you to publish provider manifests to an OCI registry by constructing an OCI artifact from the provided directory and/or files and pushing it to the specified registry.

## Usage

```bash
kubectl operator publish [OPTIONS]
```

## Options

| Flag             | Short  | Description                                                                                       |
|------------------|--------|---------------------------------------------------------------------------------------------------|
| `--artifact-url` | `-u`   | The URL of the OCI artifact to collect component manifests from. This includes the registry and optionally a version/tag. **Example**: `ttl.sh/${IMAGE_NAME}:5m` |
| `--dir`          | `-d`   | The directory containing the provider manifests. The default is the current directory (`.`). **Example**: `manifests` |
| `--file`         | `-f`   | A list of specific manifest files to include in the OCI artifact. You can specify one or more files. **Example**: `metadata.yaml`, `infrastructure-components.yaml` |

## Examples

### Publish provider manifests from a directory to the OCI registry
This command publishes all files in the `manifests` directory to the OCI registry specified in the `-u` option:
```bash
kubectl operator publish -u ttl.sh/${IMAGE_NAME}:5m -d manifests
```

### Publish specific manifest files to the OCI registry
This command publishes the `metadata.yaml` and `infrastructure-components.yaml` files to the OCI registry:
```bash
kubectl operator publish -u ttl.sh/${IMAGE_NAME}:5m -f metadata.yaml -f infrastructure-components.yaml
```

### Publish with both directory and specific files
This command combines both the directory (`manifests`) and the custom files (`metadata.yaml`, `infrastructure-components.yaml`):
```bash
kubectl operator publish -u ttl.sh/${IMAGE_NAME}:5m -d manifests -f metadata.yaml -f infrastructure-components.yaml
```

## Publishing Multiple Providers and Versions in an OCI Image

This example demonstrates how to publish three different providers (`control-plane kubeadm`, `bootstrap kubeadm`, and `infrastructure docker`) along with their versioned metadata and components files into a **single OCI image**. Each provider has two versions (`v1.9.4` and `v1.9.5`), and the corresponding metadata and components files follow versioned naming conventions.

The following layout for the directory can be used:

```bash
manifests/
├── control-plane-kubeadm-v1.9.4-metadata.yaml
├── control-plane-kubeadm-v1.9.4-components.yaml
├── bootstrap-kubeadm-v1.9.4-metadata.yaml
├── bootstrap-kubeadm-v1.9.4-components.yaml
├── infrastructure-docker-v1.9.4-metadata.yaml
├── infrastructure-docker-v1.9.4-components.yaml
├── control-plane-kubeadm-v1.9.5-metadata.yaml
├── control-plane-kubeadm-v1.9.5-components.yaml
├── bootstrap-kubeadm-v1.9.5-metadata.yaml
├── bootstrap-kubeadm-v1.9.5-components.yaml
└── infrastructure-docker-v1.9.5-metadata.yaml
└── infrastructure-docker-v1.9.5-components.yaml
```

```bash
capioperator publish -u my-registry.example.com/providers:latest -d manifests \
```

This will publish both versions (`v1.9.4` and `v1.9.5`) of each provider into single OCI image, and each version will have its corresponding metadata and component files.

### Publish with authentication
If authentication is required for the OCI registry, you can specify credentials using environment variables:
```bash
export OCI_USERNAME=myusername
export OCI_PASSWORD=mypassword
kubectl operator publish -u ttl.sh/${IMAGE_NAME}:5m -d manifests
```

## OCI Authentication

To securely authenticate with an OCI registry, the `publish` subcommand relies on environment variables for user credentials. The following environment variables are used:

- **`OCI_USERNAME`**: The username for the OCI registry.
- **`OCI_PASSWORD`**: The password associated with the username.
- **`OCI_ACCESS_TOKEN`**: A token used for authentication.
- **`OCI_REFRESH_TOKEN`**: A refresh token to obtain new access tokens.

### Example of Setting Up OCI Authentication

1. Set the environment variables with your OCI credentials:

```bash
export OCI_USERNAME=myusername
export OCI_PASSWORD=mypassword
```

2. Run the `publish` command, which will automatically use the credentials:

```bash
kubectl operator publish -u my-oci-registry.com/${IMAGE_NAME}:v0.0.1 -d manifests
```

This allows the `publish` subcommand to authenticate to the OCI registry without requiring you to manually input the credentials.