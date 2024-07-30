# Using Helm Charts

Alternatively, you can install the Cluster API operator using Helm charts:

```bash
helm repo add capi-operator https://kubernetes-sigs.github.io/cluster-api-operator
helm repo update
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system
```

#### Installing providers using Helm chart

The operator Helm chart supports a "quickstart" option for bootstrapping a management cluster. The user experience is relatively similar to [clusterctl init](https://cluster-api.sigs.k8s.io/clusterctl/commands/init.html?highlight=init#clusterctl-init):

<aside class="note warning">

<h1> Warning </h1>

The `--wait` flag is REQUIRED for the helm install command to work with providers.

</aside>

```bash
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system --set infrastructure=docker:v1.4.2  --wait --timeout 90s # core Cluster API with kubeadm bootstrap and control plane providers will also be installed
```

```bash
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system —set infrastructure="docker;azure"  --wait --timeout 90s # core Cluster API with kubeadm bootstrap and control plane providers will also be installed
```

```bash
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system —set infrastructure="capd-custom-ns:docker:v1.4.2;capz-custom-ns:azure:v1.10.0"  --wait --timeout 90s # core Cluster API with kubeadm bootstrap and control plane providers will also be installed
```

```bash
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system --set core=cluster-api:v1.4.2 --set controlPlane=kubeadm:v1.4.2 --set bootstrap=kubeadm:v1.4.2  --set infrastructure=docker:v1.4.2  --wait --timeout 90s
```

For more complex operations, please refer to our API documentation.
