# AI Agent Guidelines for cluster-api-operator

This document provides context and guidelines for AI coding assistants working with the Cluster API Operator repository.

## Project Overview

The **Cluster API Operator** is a Kubernetes Operator that manages the lifecycle of Cluster API providers within a management cluster using a declarative approach. It extends the capabilities of the `clusterctl` CLI, enabling GitOps workflows and automation.

- **Organization**: Kubernetes SIG Cluster Lifecycle
- **Module**: `sigs.k8s.io/cluster-api-operator`
- **Documentation**: https://cluster-api-operator.sigs.k8s.io

## Technology Stack

- **Language**: Go
- **Framework**: [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- **Kubernetes Libraries**: client-go, apimachinery, apiextensions-apiserver
- **Cluster API**: sigs.k8s.io/cluster-api
- **Testing**: Ginkgo/Gomega, envtest
- **Build**: Make, Docker
- **Local Development**: Tilt

## Repository Structure

```
cluster-api-operator/
├── api/v1alpha2/           # CRD type definitions and interfaces
├── cmd/                    # Main entry point and CLI plugin
├── config/                 # Kustomize manifests (CRDs, RBAC, webhooks)
├── controller/             # Public controller aliases
├── internal/
│   ├── controller/         # Controller implementations
│   ├── envtest/            # Test environment setup
│   ├── patch/              # Patch utilities
│   └── webhook/            # Admission webhook implementations
├── hack/                   # Build scripts and tools
├── test/                   # E2E tests and test framework
├── util/                   # Shared utilities
└── version/                # Version information
```

## Key Concepts

### Provider Types

The operator manages seven types of Cluster API providers:

| Type | CRD | Description |
|------|-----|-------------|
| Core | `CoreProvider` | Core Cluster API components |
| Infrastructure | `InfrastructureProvider` | Cloud/infrastructure providers (AWS, Azure, vSphere, etc.) |
| Bootstrap | `BootstrapProvider` | Node bootstrap providers (Kubeadm, etc.) |
| ControlPlane | `ControlPlaneProvider` | Control plane providers (Kubeadm, etc.) |
| Addon | `AddonProvider` | Addon providers (Helm, etc.) |
| IPAM | `IPAMProvider` | IP Address Management providers |
| RuntimeExtension | `RuntimeExtensionProvider` | Runtime extension providers |

### Generic Provider Pattern

All providers implement the `GenericProvider` interface (`api/v1alpha2/genericprovider_interfaces.go`):

```go
type GenericProvider interface {
    client.Object
    conditions.Setter
    GetSpec() ProviderSpec
    SetSpec(in ProviderSpec)
    GetStatus() ProviderStatus
    SetStatus(in ProviderStatus)
    GetType() string
    ProviderName() string
}
```

This pattern enables a single `GenericProviderReconciler` to handle all provider types.

### Reconciliation Phases

Provider reconciliation follows a phased approach (`internal/controller/phases.go`):

1. `ApplyFromCache` - Apply cached configuration if unchanged
2. `PreflightChecks` - Validate prerequisites
3. `InitializePhaseReconciler` - Set up clusterctl configuration
4. `DownloadManifests` - Fetch provider manifests (OCI/GitHub/ConfigMap)
5. `Load` - Load provider configuration
6. `Fetch` - Process YAML manifests
7. `Store` - Cache processed manifests
8. `Upgrade` - Handle version upgrades
9. `Install` - Apply provider components
10. `ReportStatus` - Update provider status
11. `Finalize` - Cleanup

## Development Guidelines

### Code Style

- Follow [Kubernetes coding conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md)
- Use `klog` for logging via controller-runtime's `ctrl.LoggerFrom(ctx)`
- Handle errors with proper wrapping using `fmt.Errorf("message: %w", err)`
- Use the `PhaseError` type for reconciliation errors with conditions

### Adding New Features

1. **API Changes**: Modify types in `api/v1alpha2/`, run `make generate manifests`
2. **Controller Changes**: Implement in `internal/controller/`
3. **Webhooks**: Add to `internal/webhook/`
4. **Tests**: Add unit tests alongside code, E2E tests in `test/e2e/`

### Testing

```bash
# Run unit tests
make test

# Run linters
make lint

# Run E2E tests
make test-e2e

# Generate mocks and deep copy
make generate
```

### Local Development with Tilt

1. Clone `cluster-api` alongside this repository
2. Configure `tilt-settings.yaml` in cluster-api:
   ```yaml
   provider_repos:
   - "../cluster-api-operator"
   enable_providers:
   - capi-operator
   enable_core_provider: false
   ```
3. Run `make tilt-up` from the cluster-api directory

### Common Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the operator binary |
| `make docker-build` | Build Docker image |
| `make test` | Run unit tests |
| `make lint` | Run linters |
| `make generate` | Generate code (deep copy, manifests) |
| `make manifests` | Generate CRD manifests |
| `make help` | Show all available targets |

## Important Patterns

### Condition Management

Use the cluster-api conditions utilities:

```go
import "sigs.k8s.io/cluster-api/util/conditions"

// Set a condition
conditions.Set(provider, metav1.Condition{
    Type:    operatorv1.ProviderInstalledCondition,
    Status:  metav1.ConditionTrue,
    Reason:  "ProviderInstalled",
    Message: "Provider installed successfully",
})
```

### Patch Helper Pattern

Always use the patch helper for updates:

```go
patchHelper, err := patch.NewHelper(provider, r.Client)
if err != nil {
    return ctrl.Result{}, err
}
defer func() {
    if err := patchHelper.Patch(ctx, provider); err != nil {
        reterr = kerrors.NewAggregate([]error{reterr, err})
    }
}()
```

### FetchConfig Sources

Providers can fetch manifests from three sources:

1. **OCI Registry**: `spec.fetchConfig.oci`
2. **GitHub URL**: `spec.fetchConfig.url`
3. **ConfigMap**: `spec.fetchConfig.selector`

## API Version

Current API version: `v1alpha2` (`operator.cluster.x-k8s.io/v1alpha2`)

## Related Projects

- [Cluster API](https://github.com/kubernetes-sigs/cluster-api) - Main Cluster API project
- [clusterctl](https://cluster-api.sigs.k8s.io/clusterctl/overview.html) - CLI tool this operator extends

## Getting Help

- Slack: [#cluster-api-operator](https://kubernetes.slack.com/archives/C030JD32R8W) on Kubernetes Slack
- Documentation: https://cluster-api-operator.sigs.k8s.io

