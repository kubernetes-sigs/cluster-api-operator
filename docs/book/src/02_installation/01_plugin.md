# Plugin installation

The `cluster-api-operator` plugin can be installed using krew, the kubectl plugin manager.

## Prerequisites

[krew][] installed on your system. See the krew installation guide for instructions.

[krew]: [https://krew.sigs.k8s.io/docs/user-guide/setup/install/]

## Steps

1. Add the cluster-api-operator plugin index to krew:
```bash
clusterctl krew index add operator https://github.com/kubernetes-sigs/cluster-api-operator.git
```

2. Install the cluster-api-operator plugin:
```bash
clusterctl krew install operator/operator
```

3. Verify the installation:
```bash
clusterctl operator
```

This should print help information for the clusterctl operator plugin.

The `cluster-api-operator` plugin is now installed and ready to use with both `kubectl` and `clusterctl`.