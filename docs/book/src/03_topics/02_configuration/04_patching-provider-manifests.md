# Patching provider manifests

Provider manifests can be patched using JSON merge patches. This can be useful when you need to modify the provider manifests that are fetched from the repository. In order to provider
manifests `spec.ManifestPatches` has to be used where an array of patches can be specified:

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  manifestPatches:
    - |
      apiVersion: v1
      kind: Service
      metadata:
        labels:
            test-label: test-value
```

More information about JSON merge patches can be found here <https://datatracker.ietf.org/doc/html/rfc7396>

There are couple of rules for the patch to match a manifest:

- The `kind` field must match the target object.
- If `apiVersion` is specified it will only be applied to matching objects.
- If `metadata.name` and `metadata.namespace` not specified, the patch will be applied to all objects of the specified kind.
- If `metadata.name` is specified, the patch will be applied to the object with the specified name. This is for cluster scoped objects.
- If both `metadata.name` and `metadata.namespace` are specified, the patch will be applied to the object with the specified name and namespace.
