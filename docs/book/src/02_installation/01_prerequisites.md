# Prerequisites

Before installing the Cluster API Operator, you must first ensure that cert-manager is installed, as the operator does not manage cert-manager installations. To install cert-manager, run the following command:

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Wait for cert-manager to be ready before proceeding.

After cert-manager is successfully installed, you can proceed installing the Cluster API operator.