# Local Development
Tilt is favoured by most Cluster API projects for local development, it offers a simple way of creating a local development environment.
Cluster API includes its own Tiltfile that can be used to run Cluster API Operator on a local Kind cluster.

## Clone the Cluster API repository

Clone the Cluster API repository in the same directory as the Cluster API Operator:

```bash
git clone https://github.com/kubernetes-sigs/cluster-api.git
```

Afterward, your folder structure should look like as follows:

```
some-folder/
├── cluster-api
└── cluster-api-operator
```

## Set up Tilt settings in `cluster-api` folder

Refer to [this guide](https://cluster-api.sigs.k8s.io/developer/core/tilt.html) to set up Tilt for Cluster API.

For our use case, you only need to configure `tilt-settings.yaml` in the `cluster-api` directory to enable the Cluster API Operator. Add the following fields to the corresponding lists in `tilt-settings.yaml`:

```yaml
provider_repos:
- "../cluster-api-operator"
enable_providers:
- capi-operator
enable_core_provider: false
```

## Run Tilt

From `cluster-api` folder run:

```bash
make docker-build-e2e # Use locally built CAPI images
make tilt-up
```

That's it! Tilt will automatically reload the deployment in your local cluster whenever you make code changes, allowing you to debug the deployed code in real time.
