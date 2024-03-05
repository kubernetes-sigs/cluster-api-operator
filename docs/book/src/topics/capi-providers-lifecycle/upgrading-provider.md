# Upgrading a Provider

To trigger an upgrade for a Cluster API provider, change the `spec.Version` field. All providers must follow the golden rule of respecting the same Cluster API contract supported by the core provider.

The operator performs the upgrade by:

1. Deleting the current provider components, while preserving CRDs, namespaces, and user objects.
2. Installing the new provider components.

Differences between the operator and `clusterctl upgrade apply` include:

- The operator upgrades one provider at a time while `clusterctl upgrade apply` upgrades a group of providers in a single operation.
- With the declarative approach, users are responsible for manually editing the Provider objects' YAML, while `clusterctl upgrade apply --contract` automatically determines the latest available versions for each provider.
