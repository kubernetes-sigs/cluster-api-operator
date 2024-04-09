# Cluster API Operator v1alpha1 compared to v1alpha2

This document provides an overview over relevant changes between Cluster API Operator API v1alpha1 and v1alpha2 for consumers of our Go API.

## Changes by Kind

The changes below affect all v1alpha1 provider kinds: `CoreProvider`, `ControlPlaneProvider`, `BootstrapPrivider` and `InfrastructureProvider`.

### API Changes

This section describes changes that were introduced in v1alpha2 API and how to update your templates to the new version.

#### ImageMeta -> imageURL conversion

In v1alpha1 we use ImageMeta object that consists of 3 parts:

- Repository (optional string): image registry (e.g., "example.com/repo")
- Name (optional string): image name (e.g., "provider-image")
- Tag (optional string): image tag (e.g., "v1.0.0")

In v1alpha2 it is just a string, which represents the URL, e.g. `example.com/repo/image-name:v1.0.0`.

Example:

v1alpha1
```yaml
spec:
 deployment:
   containers:
   - name: manager
     image:
       repository: "example.com/repo"
       name: "image-name"
       tag: "v1.0.0"
```

v1alpha2
```yaml
spec:
 deployment:
   containers:
   - name: manager
     imageURL: "example.com/repo/image-name:v1.0.0"
```

#### secretName/secretNamespace -> configSecret conversion

In v1alpha1 we have 2 separate top-level fields to point to a config secret: `secretName` and `secretNamespace`. In v1alpha2 we reworked them into an object `configSecret` that has 2 fields: `name` and `namespace`.

Example:

v1alpha1
```yaml
spec:
 secretName: azure-variables
 secretNamespace: capz-system
```

v1alpha2
```yaml
spec:
 configSecret:
   name: azure-variables
   namespace: capz-system
```
