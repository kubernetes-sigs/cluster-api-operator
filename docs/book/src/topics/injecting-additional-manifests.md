# Injecting additional manifests

It is possible to inject additional manifests when installing/upgrading a provider. This can be useful when you need to add extra RBAC resources to the provider controller, for example.
The field `AdditionalManifests` is a reference to a ConfigMap that contains additional manifests, which will be applied together with the provider components. The key for storing these manifests has to be `manifests`.
The manifests are applied only once when a certain release is installed/upgraded. If the namespace is not specified, the namespace of the provider will be used. There is no validation of the YAML content inside the ConfigMap.

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: additional-manifests
  namespace: capi-system
data:
  manifests: |
    # Additional manifests go here
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  additionalManifests:
    name: additional-manifests
```
