# Quickstart

This is a quickstart guide for getting Cluster API Operator up and running on your Kubernetes cluster.

## Prerequisites

- [Running Kubernetes cluster](https://cluster-api.sigs.k8s.io/user/quick-start#install-andor-configure-a-kubernetes-cluster).
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for interacting with the management cluster.
- [Helm](https://helm.sh/docs/intro/install/) for installing operator on the cluster (optional).

## Install and configure Cluster API Operator

### Configuring credential for cloud providers

Instead of using environment variables as clusterctl does, Cluster API Operator uses Kubernetes secrets to store credentials for cloud providers. Refer to [provider documentation](https://cluster-api.sigs.k8s.io/user/quick-start#initialization-for-common-providers) on which credentials are required.

This example uses AWS provider, but the same approach can be used for other providers.

```bash
export CREDENTIALS_SECRET_NAME="credentials-secret"
export CREDENTIALS_SECRET_NAMESPACE="default"

kubectl create secret generic "${CREDENTIALS_SECRET_NAME}" --from-literal=AWS_B64ENCODED_CREDENTIALS="${AWS_B64ENCODED_CREDENTIALS}" --namespace "${CREDENTIALS_SECRET_NAMESPACE}"
```

### Installing Cluster API Operator

#### Method 1: Apply Manifests from Release Assets

Before installing the Cluster API Operator this way, you must first ensure that cert-manager is installed, as the operator does not manage cert-manager installations. To install cert-manager, run the following command:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for cert-manager to be ready before proceeding.

After cert-manager is successfully installed, you can install the Cluster API operator directly by applying the latest release assets:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/cluster-api-operator/releases/latest/download/operator-components.yaml
```

#### Method 2: Use Helm Charts

Alternatively, you can install the Cluster API operator using Helm charts:

```bash
helm repo add capi-operator https://kubernetes-sigs.github.io/cluster-api-operator
helm repo update
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system
```

##### Installing cert-manager using Helm chart

CAPI operator Helm chart supports provisioning of cert-manager as a dependency. It is disabled by default, but you can enable it with `--set cert-manager.enabled=true` option to `helm install` command or inside of `cert-manager` section in [values.yaml](https://github.com/kubernetes-sigs/cluster-api-operator/blob/main/hack/charts/cluster-api-operator/values.yaml) file. Additionally you can define other [parameters](https://artifacthub.io/packages/helm/cert-manager/cert-manager#configuration) provided by the cert-manager chart.

##### Installing providers using Helm chart

Deploy Cluster API components with docker provider using a single command during operator installation:

```bash
helm install capi-operator capi-operator/cluster-api-operator --create-namespace -n capi-operator-system --set infrastructure=docker --set cert-manager.enabled=true --set configSecret.name=${CREDENTIALS_SECRET_NAME} --set configSecret.namespace=${CREDENTIALS_SECRET_NAMESPACE}  --wait --timeout 90s
```

Docker provider can be replaced by any provider supported by [clusterctl](https://cluster-api.sigs.k8s.io/reference/providers.html#infrastructure).