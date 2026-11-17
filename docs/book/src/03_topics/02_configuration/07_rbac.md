# RBAC

The operator ships two aggregating ClusterRoles for its provider resources:

- `capi-operator-aggregate-view-role` aggregates read access (`get`, `list`, `watch`, including `/status`) into the builtin `view`, `edit` and `admin` ClusterRoles.
- `capi-operator-aggregate-admin-role` aggregates write access (`create`, `update`, `patch`, `delete`, `deletecollection`) into the builtin `admin` ClusterRole only. Users with only `edit` can read the provider kinds but not modify them, because provider resources make the operator install cluster-wide manifests.

They cover all operator provider resources:

- `coreproviders`
- `infrastructureproviders`
- `bootstrapproviders`
- `controlplaneproviders`
- `addonproviders`
- `ipamproviders`
- `runtimeextensionproviders`

Provider resources only reference credentials through Secrets (for example `spec.configSecret`), so they are readable by `view`.
Keep credentials out of inline fields such as `spec.deployment.containers[].env[].value`; use `valueFrom` instead.
