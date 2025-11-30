# Patching provider manifests

Provider manifests can be patched to customize the resources that are fetched from the provider repository before they are applied to the cluster. There are two supported mechanisms for patching provider manifests:

* `spec.manifestPatches` - (legacy) supports only JSON merge patches (RFC 7396).
* `spec.patches` - generic patches with explicit targeting and support for both strategic merge and RFC 6902 JSON patches.

> ⚠️ **Note:** `spec.manifestPatches` and `spec.patches` are mutually exclusive. You must specify at most one of them.

---

## Patching using `manifestPatches` (legacy)

To modify provider manifests, use `spec.manifestPatches` to specify an array of patches.

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

## Patching using `patches`

The `spec.patches` field provides a more flexible and expressive way to patch provider manifests. It allows:

* Explicit targeting using Group / Version / Kind / Name / Namespace / Label selectors.
* Support for strategic merge patch and RFC 6902 JSON patches.
* Clear separation between what to patch and where to apply it.
* Each entry in `spec.patches` consists of a patch and a target.

```yaml
---
# Strategic merge patch
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  patches:
    - patch: |
        apiVersion: v1
        kind: Service
        metadata:
          labels:
            test-label: test-value
      target:
        kind: Service
---
# RFC 6902 JSON Patch
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  patches:
    - patch: |
        - op: add
          path: /spec/template/spec/containers/0/args/-
          value: --additional-sync-machine-labels=topology.kubernetes.io/.*
      target:
        group: apps
        version: v1
        kind: Deployment
        name: capi-controller-manager
        namespace: capi-system
```

### Target Matching

A patch in spec.patches is applied to a rendered manifest if it matches the target selector.

The following fields may be used to select target objects:

* `group` – API group (for example: apps).
* `version` – API version (for example: v1).
* `kind` – Kind of the object.
* `name` – Name of the object.
* `namespace` – Namespace of the object.
* `labelSelector` – Label selector expression as defined by Kubernetes.

#### Matching behavior

- If target is omitted, the patch is applied to all rendered objects.
- If only kind is specified, the patch is applied to all objects of that kind.
- If name is specified, the patch is applied only to objects with that name.
- If both name and namespace are specified, the patch is applied only to the object with that name and namespace.
- If labelSelector is specified, the patch is applied only to objects whose labels match the selector.

**All specified fields must match for the patch to be applied.**
