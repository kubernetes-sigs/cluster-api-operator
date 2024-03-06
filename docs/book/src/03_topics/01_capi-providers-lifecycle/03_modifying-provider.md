# Modifying a Provider

In addition to changing a provider version (upgrades), the operator supports modifying other provider fields such as controller flags and variables. This can be achieved through `kubectl edit` or `kubectl apply` to the provider object.

The operation works similarly to upgrades: The current provider instance is deleted while preserving CRDs, namespaces, and user objects. Then, a new provider instance with the updated flags/variables is installed.

**Note**: `clusterctl` currently does not support this operation.
