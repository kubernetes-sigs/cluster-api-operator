# Plugin installation

The `cluster-api-operator` plugin can be installed using krew, the kubectl plugin manager.

## Prerequisites

[krew][] installed on your system. See the krew installation guide for instructions.

[krew]: [https://krew.sigs.k8s.io/docs/user-guide/setup/install/]

## Steps

1. Add the cluster-api-operator plugin index to krew:
```bash
kubectl krew index add operator https://github.com/kubernetes-sigs/cluster-api-operator.git
```

2. Install the cluster-api-operator plugin:
```bash
kubectl krew install operator/clusterctl-operator
```

3. Verify the installation:
```bash
kubectl operator
```

This should print help information for the kubectl operator plugin.

The `cluster-api-operator` plugin is now installed and ready to use with `kubectl`.

### Optionally: installing as a `clusterctl` plugin
Typically the plugin is installed under `~/.krew/bin/kubectl-operator`, which would be present under your `$PATH` after correct `krew` installation. If you want to use plugin with `clusterctl`, you need to rename this file to be prefixed with `clusterctl-` instead, like so:
```bash
cp ~/.krew/bin/kubectl-operator ~/.krew/bin/clusterctl-operator
```

After that plugin is available to use as a `clusterctl` plugin:
```bash
clusterctl operator --help
```

## Upgrade

To upgrade your plugin with the new release of `cluster-api-operator` you will need to run:

```bash
kubectl krew upgrade
```