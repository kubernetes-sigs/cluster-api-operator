# E2E Tests

## Overview

The end-to-end (E2E) test suite validates the full lifecycle of the Cluster API Operator in a real Kubernetes cluster. Tests cover provider creation, upgrade, downgrade, deletion, air-gapped installations, OCI registry support, compressed manifests, and Helm chart rendering.

## Running E2E Tests

### Quick Start (Local)

```bash
make test-e2e-local
```

This creates a local Kind cluster, deploys cert-manager and the operator, and runs the full E2E suite.

### Using an Existing Cluster

```bash
USE_EXISTING_CLUSTER=true make test-e2e
```

### Running Specific Tests

Use Ginkgo's `--focus` flag to run a subset of tests:

```bash
# Run only air-gapped tests
make test-e2e GINKGO_ARGS="--focus='air gapped'"

# Run only CoreProvider tests
make test-e2e GINKGO_ARGS="--focus='CoreProvider'"
```

### Skipping Cleanup

For debugging failed tests, set `SKIP_CLEANUP=true` to preserve cluster state:

```bash
SKIP_CLEANUP=true make test-e2e-local
```

## Test Suite Structure

```
test/e2e/
├── e2e_suite_test.go              # Suite setup, Kind cluster management, cert-manager
├── helpers_test.go                # Shared test utilities and helper functions
├── minimal_configuration_test.go  # Core provider lifecycle tests (create/upgrade/delete)
├── air_gapped_test.go             # ConfigMap-based air-gapped installation tests
├── compressed_manifests_test.go   # Large manifest compression via OCI
├── helm_test.go                   # Helm chart rendering and golden-file tests
├── config/                        # E2E configuration YAML files
├── resources/                     # Test resource manifests
└── doc.go                         # Package documentation
```

### Test Files

| File | Tests | Description |
|------|-------|-------------|
| `minimal_configuration_test.go` | 11 | Provider create, upgrade, downgrade, delete for all 7 types; OCI fetching; manifest patches |
| `air_gapped_test.go` | 3 | ConfigMap-based install/upgrade without network access |
| `compressed_manifests_test.go` | 4 | Large OCI manifests exceeding ConfigMap 1MB limit |
| `helm_test.go` | 16 | Helm chart install + 15 golden-file template comparison tests |

## Test Framework

The E2E tests use:

- **[Ginkgo v2](https://onsi.github.io/ginkgo/)** — BDD test framework
- **[Gomega](https://onsi.github.io/gomega/)** — Matcher library with `Eventually`/`Consistently` support
- **[CAPI test framework](https://pkg.go.dev/sigs.k8s.io/cluster-api/test/framework)** — Kubernetes cluster management utilities
- **Custom framework** (`test/framework/`) — Operator-specific helpers (`HaveStatusConditionsTrue`, `For().In().ToSatisfy()`)

### Key Patterns

#### Condition Checking

Use the `HaveStatusConditionsTrue` helper to verify provider conditions:

```go
HaveStatusConditionsTrue(
    provider,
    operatorv1.PreflightCheckCondition,
    operatorv1.ProviderInstalledCondition,
)
```

#### Eventually / Consistently

Always use `Eventually` for async operations (provider creation, deployment readiness) and `Consistently` to assert that a state holds over time:

```go
// Wait for provider to become ready
Eventually(func() bool {
    // ... check condition
}, e2eConfig.GetIntervals(...)...).Should(BeTrue())

// Verify condition stays true
Consistently(func() bool {
    // ... check condition
}, e2eConfig.GetIntervals(...)...).Should(BeTrue())
```

#### Configurable Intervals

Test timeouts and poll intervals are configured in `config/` YAML files, not hard-coded:

```yaml
intervals:
  default/wait-providers: ["5m", "10s"]
  default/wait-controllers: ["3m", "10s"]
```

Access them with:

```go
e2eConfig.GetIntervals("default", "wait-providers")
```

## Writing New E2E Tests

### 1. Add a Test File

Create a new file in `test/e2e/` with the `e2e` build tag:

```go
//go:build e2e

package e2e

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
    . "sigs.k8s.io/cluster-api-operator/test/framework"
)
```

### 2. Use Ginkgo Containers

Structure tests with `Describe`, `Context`, and `It`:

```go
var _ = Describe("My Feature", func() {
    It("should do something", func() {
        // Test implementation
    })
})
```

For ordered tests that share state, use `Ordered`:

```go
var _ = Describe("Sequential tests", Ordered, func() {
    It("step 1", func() { /* ... */ })
    It("step 2", func() { /* ... */ })
})
```

### 3. Create Provider Resources

Use the standard pattern from existing tests:

```go
coreProvider := &operatorv1.CoreProvider{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "cluster-api",
        Namespace: operatorNamespace,
    },
    Spec: operatorv1.CoreProviderSpec{
        ProviderSpec: operatorv1.ProviderSpec{
            Version: "v1.9.0",
        },
    },
}

Expect(bootstrapClusterProxy.GetClient().Create(ctx, coreProvider)).To(Succeed())
```

### 4. Wait for Conditions

```go
Eventually(
    For(coreProvider).
        In(bootstrapClusterProxy.GetClient()).
        ToSatisfy(
            HaveStatusConditionsTrue(
                coreProvider,
                operatorv1.PreflightCheckCondition,
                operatorv1.ProviderInstalledCondition,
            ),
        ),
    e2eConfig.GetIntervals("default", "wait-providers")...,
).Should(BeTrue())
```

### 5. Clean Up Resources

Always clean up after tests to avoid interfering with other specs:

```go
AfterEach(func() {
    Expect(bootstrapClusterProxy.GetClient().Delete(ctx, coreProvider)).To(Succeed())
    // Wait for deletion to complete
    Eventually(func() bool {
        err := bootstrapClusterProxy.GetClient().Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)
        return apierrors.IsNotFound(err)
    }, e2eConfig.GetIntervals("default", "wait-providers")...).Should(BeTrue())
})
```

### 6. Add Golden Files (Helm Tests)

For Helm template tests, add expected output in `test/e2e/resources/` and compare:

```go
rendered := helmTemplate(chartPath, releaseName, namespace, values)
expected := loadGoldenFile("resources/expected-output.yaml")
Expect(rendered).To(Equal(expected))
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `USE_EXISTING_CLUSTER` | Use existing cluster instead of Kind | `false` |
| `SKIP_CLEANUP` | Skip resource cleanup after tests | `false` |
| `E2E_CONFIG_PATH` | Path to E2E config YAML | `test/e2e/config/operator.yaml` |
| `ARTIFACTS_FOLDER` | Folder for test artifacts/logs | `_artifacts` |
| `GINKGO_ARGS` | Additional Ginkgo CLI arguments | — |

## Debugging Tips

1. **Preserve cluster state**: Use `SKIP_CLEANUP=true` to keep resources after failure.
2. **Collect logs**: Artifacts are stored in the `ARTIFACTS_FOLDER` directory including pod logs and cluster state.
3. **Run focused tests**: Use `--focus` to isolate failing tests.
4. **Check provider conditions**: When a provider isn't becoming ready, examine its `.status.conditions` for error details.
5. **Inspect deployments**: Provider components are deployed in the provider's namespace; check controller-manager pod logs.

## Compatibility Notice

This package is not subject to deprecation notices or compatibility guarantees.

- Breaking changes are likely. External providers using this package should update to the latest API changes when updating Cluster API Operator. Maintainers and contributors must give notice in release notes when a breaking change happens.
