# Examples of API Usage

In this section we provide some concrete examples of CAPI Operator API usage for various use-cases.

1. As an admin, I want to install the aws infrastructure provider with specific controller flags.

```yaml
apiVersion: v1
kind: Secret
metadata:
 name: aws-variables
 namespace: capa-system
type: Opaque
data:
 AWS_B64ENCODED_CREDENTIALS: ...
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
 name: aws
 namespace: capa-system
spec:
 version: v2.1.4
 configSecret:
   name: aws-variables
 manager:
   # These top level controller manager flags, supported by all the providers.
   # These flags come with sensible defaults, thus requiring no or minimal
   # changes for the most common scenarios.
   metrics:
    bindAddress: ":8181"
   syncPeriod: "500s"
 fetchConfig:
   url: https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases
 deployment:
   containers:
   - name: manager
     args:
      # These are controller flags that are specific to a provider; usage
      # is reserved for advanced scenarios only.
      "--awscluster-concurrency": "12"
      "--awsmachine-concurrency": "11"
```

2. As an admin, I want to install aws infrastructure provider but override the container image of the CAPA deployment.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
 name: aws
 namespace: capa-system
spec:
 version: v2.1.4
 configSecret:
   name: aws-variables
 deployment:
   containers:
   - name: manager
     imageUrl: "gcr.io/myregistry/capa-controller:v2.1.4-foo"
```

3. As an admin, I want to change the resource limits for the manager pod in my control plane provider deployment.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: ControlPlaneProvider
metadata:
 name: kubeadm
 namespace: capi-kubeadm-control-plane-system
spec:
 version: v1.4.3
 configSecret: 
   name: capi-variables
 deployment:
   containers:
   - name: manager
     resources:
       limits:
         cpu: 100m
         memory: 30Mi
       requests:
         cpu: 100m
         memory: 20Mi
```

4. As an admin, I would like to fetch my azure provider components from a specific repository which is not the default.

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
 name: myazure
 namespace: capz-system
spec:
 version: v1.9.3
 configSecret:
   name: azure-variables
 fetchConfig:
   url: https://github.com/myorg/awesome-azure-provider/releases

```

5. As an admin, I would like to use the default fetch configurations by simply specifying the expected Cluster API provider names such as `aws`, `vsphere`, `azure`, `kubeadm`, `talos`, or `cluster-api` instead of having to explicitly specify the fetch configuration. In the example below, since we are using 'vsphere' as the name of the InfrastructureProvider the operator will fetch it's configuration from `url: https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/releases` by default.

See more examples in the [air-gapped environment section](./01_air-gapped-environtment.md)

```yaml
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata:
 name: vsphere
 namespace: capv-system
spec:
 version: v1.6.1
 configSecret:
   name: vsphere-variables
```
