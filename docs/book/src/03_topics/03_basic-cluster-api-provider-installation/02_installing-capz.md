# Installing Azure Infrastructure Provider

Next, install [Azure Infrastructure Provider](https://capz.sigs.k8s.io/). Before that ensure that `capz-system` namespace exists.

Since the provider requires variables to be set, create a secret containing them in the same namespace as the provider. It is also recommended to include a `github-token` in the secret. This token is used to fetch the provider repository, and it is required for the provider to be installed. The operator may exceed the rate limit of the GitHub API without the token. Like [clusterctl](https://cluster-api.sigs.k8s.io/clusterctl/overview.html?highlight=github_token#avoiding-github-rate-limiting), the token needs only the `repo` scope.

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: azure-variables
  namespace: capz-system
type: Opaque
stringData:
  AZURE_CLIENT_ID_B64: Zm9vCg==
  AZURE_CLIENT_SECRET_B64: Zm9vCg==
  AZURE_SUBSCRIPTION_ID_B64: Zm9vCg==
  AZURE_TENANT_ID_B64: Zm9vCg==
  github-token: ghp_fff
---
apiVersion: operator.cluster.x-k8s.io/v1alpha1
kind: InfrastructureProvider
metadata:
 name: azure
 namespace: capz-system
spec:
 version: v1.9.3
 configSecret:
   name: azure-variables
```
