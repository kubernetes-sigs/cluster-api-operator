# Deleting providers

To remove all installed providers and all related kubernetes objects just delete the following CRs:

```bash
kubectl delete coreprovider --all --all-namespaces
kubectl delete infrastructureprovider --all --all-namespaces
kubectl delete bootstrapprovider --all --all-namespaces
kubectl delete controlplaneprovider --all --all-namespaces
kubectl delete ipamprovider --all --all-namespaces
kubectl delete addonprovider --all --all-namespaces
```
