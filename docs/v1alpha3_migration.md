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
    - name: manager
      args:
        "--max-concurrent-reconciles": "10"
        "--v": "2"
        "--feature-gates": "FeatureA=true"
```

### 3. AdditionalDeployments Changes

The `AdditionalDeployments` type has been simplified:
- **v1alpha2**: `map[string]AdditionalDeployments` where `AdditionalDeployments` is a struct with `Manager` and `Deployment` fields
- **v1alpha3**: `map[string]DeploymentSpec` - directly maps to deployment specifications without the wrapper struct

#### v1alpha2 (Old)
```yaml
additionalDeployments:
  webhook:
    manager:
      verbosity: 2
      metrics:
        bindAddress: ":9090"
    deployment:
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
      args:
        "--v": "2"
        "--metrics-bind-addr": ":9090"
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

1. **v1alpha2 → v1alpha3**: The `ManagerSpec` fields are automatically converted to container args
2. **v1alpha3 → v1alpha2**: Container args are parsed and reconstructed into a `ManagerSpec` structure
3. **Round-trip guarantee**: Converting from v1alpha2 to v1alpha3 and back preserves all settings

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

See `v1alpha3_example.yaml` for complete examples of both v1alpha2 and v1alpha3 formats.
