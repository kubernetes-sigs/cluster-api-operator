# Using Manifests from Release Assets

Before installing the Cluster API Operator this way, you must first ensure that cert-manager is installed, as the operator does not manage cert-manager installations. To install cert-manager, run the following command:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for cert-manager to be ready before proceeding.

After cert-manager is successfully installed, you can install the Cluster API operator directly by applying the latest release assets:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/cluster-api-operator/releases/latest/download/operator-components.yaml
```
