# Deleting a Provider

To remove the installed providers and all related kubernetes objects just delete the following CRs:

```bash
kubectl delete infrastructureprovider azure
kubectl delete coreprovider cluster-api
```
