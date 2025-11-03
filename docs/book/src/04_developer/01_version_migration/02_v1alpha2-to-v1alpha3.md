# Migration Guide: v1alpha2 to v1alpha3

## Overview

The v1alpha3 API version introduces significant simplifications by removing the `ManagerSpec` type and consolidating all provider configuration directly into `ContainerSpec.Args`. This change makes the API more flexible and easier to understand.

## Key Changes

### 1. Removed Types
- `ManagerSpec` - All fields moved to container args
- `ControllerManagerConfiguration` - Configuration now in args
- `ControllerConfigurationSpec` - Controller settings now in args
- `ControllerMetrics` - Metrics configuration now in args  
- `ControllerHealth` - Health probe configuration now in args
- `ControllerWebhook` - Webhook configuration now in args
- `AdditionalDeployments` struct - Replaced with direct `DeploymentSpec`

### 2. ProviderSpec Changes

During conversion from v1alpha2 to v1alpha3, the `ManagerSpec` is converted to a container with an **empty name** (`""`). The code then determines the correct container name to apply these changes to at runtime using the following logic:
1. First, it checks if a default container is specified in the deployment's annotations (`kubectl.kubernetes.io/default-container`)
2. Then, it looks for a container named "manager"
3. As a last resort, it uses the first container in the deployment

**Important**: Even if the container with empty name is specified first in the container list, its changes will be applied **after** all other containers. This ensures that explicit container configurations take precedence over the converted ManagerSpec settings.

#### v1alpha2 (Old)
```yaml
spec:
  manager:
    maxConcurrentReconciles: 10
    verbosity: 2
    featureGates:
      FeatureA: true
  deployment:
    containers:
    - name: manager
```

#### v1alpha3 (New)
```yaml
spec:
  deployment:
    containers:
    - name: "" # Empty name - the code will determine the correct container
      args:
        "--max-concurrent-reconciles": "10"
        "--v": "2"
        "--feature-gates": "FeatureA=true"
```

### 3. AdditionalDeployments Changes

The `AdditionalDeployments` type has been simplified:
- **v1alpha2**: `map[string]AdditionalDeployments` where `AdditionalDeployments` is a struct with `Manager` and `Deployment` fields
- **v1alpha3**: `map[string]DeploymentSpec` - directly maps to deployment specifications without the wrapper struct

**Important Note**: During conversion from v1alpha2 to v1alpha3, the `ManagerSpec` fields in `additionalDeployments` are **NOT** converted to container args. This is because additional deployments are not Cluster API providers, and therefore the `ManagerSpec` configuration is not applicable to them. The `Manager` field is simply dropped during conversion, and only the `Deployment` field is preserved.

#### v1alpha2 (Old)
```yaml
additionalDeployments:
  webhook:
    manager:  # These settings are dropped during conversion
      verbosity: 2
      metrics:
        bindAddress: ":9090"
    deployment:  # Only this is preserved
      replicas: 2
      containers:
      - name: webhook
```

#### v1alpha3 (New)
```yaml
additionalDeployments:
  webhook:  # Directly a DeploymentSpec, no wrapper struct
    replicas: 2
    containers:
    - name: webhook
      # Note: manager settings are not converted to args
      # Users must manually add args if needed
```

## Argument Mappings

| v1alpha2 ManagerSpec Field | v1alpha3 Container Arg |
|---|---|
| `maxConcurrentReconciles` | `--max-concurrent-reconciles` |
| `cacheNamespace` | `--namespace` |
| `health.healthProbeBindAddress` | `--health-addr` |
| `leaderElection.leaderElect` | `--leader-elect` |
| `leaderElection.resourceNamespace/resourceName` | `--leader-election-id=<namespace>/<name>` |
| `leaderElection.leaseDuration` | `--leader-elect-lease-duration` |
| `leaderElection.renewDeadline` | `--leader-elect-renew-deadline` |
| `leaderElection.retryPeriod` | `--leader-elect-retry-period` |
| `metrics.bindAddress` | `--metrics-bind-addr` |
| `metrics.diagnosticsAddress` | `--diagnostics-address` |
| `metrics.insecureDiagnostics` | `--insecure-diagnostics` |
| `webhook.host` | `--webhook-host` |
| `webhook.port` | `--webhook-port` |
| `webhook.certDir` | `--webhook-cert-dir` |
| `syncPeriod` | `--sync-period` |
| `profilerAddress` | `--profiler-address` |
| `verbosity` | `--v` |
| `featureGates` | `--feature-gates` |
| `controller.groupKindConcurrency.<resource>` | `--<resource>-concurrency` |
| `additionalArgs` | Merged directly into args |

### Special Cases

1. **LivenessEndpointName** and **ReadinessEndpointName**: These are not converted to args but should be configured by modifying the container's probe paths directly.

2. **GroupKindConcurrency**: Each entry becomes a separate arg. For example:
   - `controller.groupKindConcurrency.Cluster: 10` → `--cluster-concurrency=10`
   - `controller.groupKindConcurrency.Machine: 5` → `--machine-concurrency=5`

3. **Feature Gates**: Multiple feature gates are combined into a single comma-separated arg:
   - `featureGates: {FeatureA: true, FeatureB: false}` → `--feature-gates=FeatureA=true,FeatureB=false`

## Conversion Behavior

The operator includes automatic conversion between v1alpha2 and v1alpha3:

1. **v1alpha2 → v1alpha3**: 
   - The `ManagerSpec` fields are automatically converted to container args in a container with **empty name** (`""`)
   - The runtime code determines which container to apply these settings to (see logic in section 2)
   - Container with empty name is processed **after** all other containers, ensuring explicit configurations take precedence
   - For `additionalDeployments`, the `ManagerSpec` is **not** converted - only the `Deployment` field is preserved

2. **v1alpha3 → v1alpha2**: 
   - Container args are parsed and reconstructed into a `ManagerSpec` structure
   - The container with empty name (if present) is interpreted as the manager configuration

3. **Round-trip guarantee**: Converting from v1alpha2 to v1alpha3 and back preserves all settings for the main provider deployment (though `additionalDeployments` manager settings are lost)

## Hub Version

v1alpha3 is marked as the hub version and storage version. This means:
- All versions are converted to v1alpha3 internally
- v1alpha3 is stored in etcd
- Conversions happen automatically when accessing different API versions

## Benefits

1. **Simplicity**: Single place for all provider configuration, no wrapper structs
2. **Flexibility**: Easy to add custom provider-specific flags
3. **Transparency**: Direct mapping to container arguments
4. **Compatibility**: Automatic conversion maintains backward compatibility

## Example Migration

Here are complete examples of both v1alpha2 and v1alpha3 formats.

```yaml
# Example of v1alpha3 CoreProvider without ManagerSpec
# All manager configuration is now in containerSpec.args
apiVersion: operator.cluster.x-k8s.io/v1alpha3
kind: CoreProvider
metadata:
  name: cluster-api
  namespace: capi-system
spec:
  version: v1.5.0
  deployment:
    replicas: 1
    containers:
    # When converted from v1alpha2, the ManagerSpec becomes a container with empty name
    - name: ""  # Empty name - runtime will determine the target container
      args:
        # These args replace the old ManagerSpec fields
        "--max-concurrent-reconciles": "10"
        "--namespace": "capi-system"
        "--health-addr": ":8081"
        "--metrics-bind-addr": ":8080"
        "--leader-elect": "true"
        "--leader-election-id": "capi-system/cluster-api-controller-leader"
        "--sync-period": "30s"
        "--v": "2"
        "--feature-gates": "ClusterTopology=true,MachinePool=false"
        "--webhook-host": "0.0.0.0"
        "--webhook-port": "9443"
        "--profiler-address": "localhost:6060"
        # Concurrency settings for different resources
        "--cluster-concurrency": "10"
        "--machine-concurrency": "5"
        # Custom args
        "--custom-flag": "custom-value"
  # Example of additionalDeployments - now directly maps to DeploymentSpec
  # No longer wrapped in an extra struct, just direct DeploymentSpec values
  # NOTE: ManagerSpec from v1alpha2 is NOT converted for additional deployments
  additionalDeployments:
    webhook:
      replicas: 2
      containers:
      - name: webhook
        # Users must manually specify args - not converted from v1alpha2 ManagerSpec
---
# Example of v1alpha2 CoreProvider with ManagerSpec (old style)
# This will be automatically converted to v1alpha3 format
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata:
  name: cluster-api-v2
  namespace: capi-system
spec:
  version: v1.5.0
  manager:
    maxConcurrentReconciles: 10
    cacheNamespace: capi-system
    health:
      healthProbeBindAddress: ":8081"
    metrics:
      bindAddress: ":8080"
    leaderElection:
      leaderElect: true
      resourceNamespace: capi-system
      resourceName: cluster-api-controller-leader
    syncPeriod: 30s
    verbosity: 2
    featureGates:
      ClusterTopology: true
      MachinePool: false
    webhook:
      host: "0.0.0.0"
      port: 9443
    profilerAddress: "localhost:6060"
    controller:
      groupKindConcurrency:
        cluster: 10
        machine: 5
    additionalArgs:
      "--custom-flag": "custom-value"
  deployment:
    replicas: 1
    containers:
    - name: manager
  # Example of additionalDeployments in v1alpha2 - includes both Manager and Deployment
  # IMPORTANT: The manager field here will be IGNORED during conversion to v1alpha3
  additionalDeployments:
    webhook:
      manager:  # This entire section is dropped during conversion
        verbosity: 2
        metrics:
          bindAddress: ":9090"
        webhook:
          port: 8443
      deployment:  # Only this section is preserved in v1alpha3
        replicas: 2
        containers:
        - name: webhook
```
