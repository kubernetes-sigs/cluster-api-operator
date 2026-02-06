# Profiling

This section explains how to set up and use debugging endpoints like pprof for the Cluster API Operator.

### Configuring Helm Values

Profiling is enabled by default but some values can be customized. You can set the following values in your `values.yaml` file:

```yaml
profilerAddress: ":6060"
contentionProfiling: true
```

Install with these custom values using [Helm chart installation methods](../installation/helm-chart-installation.md)

### Enabling Port-Forwarding

To access the pprof server on your local machine, run:

```bash
kubectl port-forward deployment/capi-operator -n <namespace> 6060
```

This will forward port 6060 from the container to your local machine.

### Running pprof Commands

With port-forwarding in place, you can run pprof commands like this:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```
