# Installing a Provider

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
