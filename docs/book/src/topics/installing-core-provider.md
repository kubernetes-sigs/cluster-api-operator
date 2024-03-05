# Installing the CoreProvider

The first step is to install the CoreProvider, which is responsible for managing the Cluster API CRDs and the Cluster API controller.

You can utilize any existing namespace for providers in your Kubernetes operator. However, before creating a provider object, make sure the specified namespace has been created. In the example below, we use the `capi-system` namespace. You can create this namespace through either the Command Line Interface (CLI) by running `kubectl create namespace capi-system`, or by using the declarative approach described in the [official Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/namespaces-walkthrough/#create-new-namespaces).

*Example:*

```yaml
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  version: v1.4.3
```

**Note:** Only one CoreProvider can be installed at the same time on a single cluster.
