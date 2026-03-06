# Contributing Guidelines

Welcome to Kubernetes. We are excited about the prospect of you joining our [community](https://git.k8s.io/community)! The Kubernetes community abides by the CNCF [code of conduct](code-of-conduct.md). Here is an excerpt:

_As contributors and maintainers of this project, and in the interest of fostering an open and welcoming community, we pledge to respect all people who contribute through reporting issues, posting feature requests, updating documentation, submitting pull requests or patches, and other activities._

## Getting Started

We have full documentation on how to get started contributing here:

- [Contributor License Agreement](https://git.k8s.io/community/CLA.md) Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests
- [Kubernetes Contributor Guide](https://git.k8s.io/community/contributors/guide) - Main contributor documentation, or you can just jump directly to the [contributing section](https://git.k8s.io/community/contributors/guide#contributing)
- [Contributor Cheat Sheet](https://git.k8s.io/community/contributors/guide/contributor-cheatsheet) - Common resources for existing developers

## Development Setup

### Prerequisites

- Go (see `Makefile` for the required version)
- Docker
- `make`
- Access to a Kubernetes cluster (for E2E tests)

### Building

```bash
# Build the operator binary
make build

# Build the Docker image
make docker-build
```

### Running Tests

```bash
# Run unit tests
make test

# Run linters
make lint

# Run E2E tests (requires a cluster)
make test-e2e
```

### Code Generation

After modifying API types in `api/v1alpha2/`, regenerate code and manifests:

```bash
make generate manifests
```

### Local Development with Tilt

For a fast inner-loop development cycle using [Tilt](https://tilt.dev/):

1. Clone [cluster-api](https://github.com/kubernetes-sigs/cluster-api) alongside this repository
2. Configure `tilt-settings.yaml` in the cluster-api directory:
   ```yaml
   provider_repos:
   - "../cluster-api-operator"
   enable_providers:
   - capi-operator
   enable_core_provider: false
   ```
3. Run `make tilt-up` from the cluster-api directory

See [docs/local-development.md](docs/local-development.md) for more details.

## Making Changes

### Repository Structure

| Directory | Description |
|-----------|-------------|
| `api/v1alpha2/` | CRD type definitions and interfaces |
| `internal/controller/` | Controller implementations |
| `internal/webhook/` | Admission webhook implementations |
| `config/` | Kustomize manifests (CRDs, RBAC, webhooks) |
| `test/e2e/` | End-to-end tests |
| `util/` | Shared utilities |

### Code Style

- Follow [Kubernetes coding conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md)
- Use `ctrl.LoggerFrom(ctx)` for structured logging
- Wrap errors with `fmt.Errorf("context: %w", err)`
- All new code must pass `make lint`

### Pull Request Process

1. Fork the repository and create a feature branch
2. Write tests for new functionality
3. Ensure `make lint` and `make test` pass locally
4. PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `fix:`, `feat:`, `docs:`)
5. PRs require at least one approving review from a maintainer listed in [OWNERS](OWNERS)
6. CI must pass before merge (linting, unit tests, E2E)

## Mentorship

- [Mentoring Initiatives](https://git.k8s.io/community/mentoring) - We have a diverse set of mentorship programs available that are always looking for volunteers!

## Contact Information

- [Slack: #cluster-api-operator](https://kubernetes.slack.com/archives/C030JD32R8W) on Kubernetes Slack
- [Documentation](https://cluster-api-operator.sigs.k8s.io)
