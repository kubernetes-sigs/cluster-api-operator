# Local Development
Tilt is favoured by most Cluster API projects for local development, it offers a simple way of creating a local development environment.

## Create a kind cluster
We'll need to create a kind cluster to deploy the operator to. This cluster will be used to deploy `cluster-api-operator` to, along with all dependencies such as `cert-manager`
```bash
kind create cluster
```

## Run Tilt
Once the cluster is live and you've confirmed you're using the correct context, you can simply run:
```bash
tilt up
```

That's it! Tilt will automatically reload the deployment to your local cluster every time you make a code change.
