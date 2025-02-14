# Local Development
Tilt is favoured by most Cluster API projects for local development, it offers a simple way of creating a local development environment.
Cluster API includes its own Tiltfile that can be used to run Cluster API Operator on a local Kind cluster.

## Clone the Cluster API repository

Clone the Cluster API repository on the same level folder where Cluster API Operator located:

```bash
$ git clone https://github.com/kubernetes-sigs/cluster-api.git
```

Afterward your folder structure should look like this:

```
some-folder/
├── cluster-api
└── cluster-api-operator
```

## Set up Tilt settings in `cluster-api` folder

Refer to [this guide](https://cluster-api.sigs.k8s.io/developer/core/tilt.html) to set up Tilt for Cluster API.

In particular, for our purposes we only need to set up `tilt-settings.yaml` in Cluster API to enable Cluster API Operator. Add the following fields to the lists in `tilt-settings.yaml`:

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
$ make tilt-up
```

That's it! Tilt will automatically reload the deployment to your local cluster every time you make a code change and able to debug deployed code.
