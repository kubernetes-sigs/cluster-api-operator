# Provider Spec

1. `ProviderSpec`: desired state of the Provider, consisting of:
   - Version (string): provider version (e.g., "v0.1.0")
   - Manager (optional ManagerSpec): controller manager properties for the provider
   - Deployment (optional DeploymentSpec): deployment properties for the provider
   - ConfigSecret (optional SecretReference): reference to the config secret
   - FetchConfig (optional FetchConfiguration): how the operator will fetch components and metadata

   YAML example:

   ```yaml
   ...
   spec:
    version: "v0.1.0"
    manager:
      maxConcurrentReconciles: 5
    deployment:
      replicas: 1
    configSecret:
      name: "provider-secret"
    fetchConfig:
      url: "https://github.com/owner/repo/releases"
   ...
   ```

2. `ManagerSpec`: controller manager properties for the provider, consisting of:
   - ProfilerAddress (optional string): pprof profiler bind address (e.g., "localhost:6060")
   - MaxConcurrentReconciles (optional int): maximum number of concurrent reconciles
   - Verbosity (optional int): logs verbosity
   - FeatureGates (optional map[string]bool): provider specific feature flags

   YAML example:

   ```yaml
   ...
   spec:
    manager:
      profilerAddress: "localhost:6060"
      maxConcurrentReconciles: 5
      verbosity: 1
      featureGates:
        FeatureA: true
        FeatureB: false
   ...
   ```

3. `DeploymentSpec`: deployment properties for the provider, consisting of:
   - Replicas (optional int): number of desired pods
   - NodeSelector (optional map[string]string): node label selector
   - Tolerations (optional []corev1.Toleration): pod tolerations
   - Affinity (optional corev1.Affinity): pod scheduling constraints
   - Containers (optional []ContainerSpec): list of deployment containers
   - ServiceAccountName (optional string): pod service account
   - ImagePullSecrets (optional []corev1.LocalObjectReference): list of image pull secrets specified in the Deployment

   YAML example:

   ```yaml
   ...
   spec:
     deployment:
       replicas: 2
       nodeSelector:
         disktype: ssd
       tolerations:
       - key: "example"
         operator: "Exists"
         effect: "NoSchedule"
       affinity:
         nodeAffinity:
           requiredDuringSchedulingIgnoredDuringExecution:
             nodeSelectorTerms:
             - matchExpressions:
               - key: "example"
                 operator: "In"
                 values:
                 - "true"
       containers:
         - name: "containerA"
           imageUrl: "example.com/repo/image-name:v1.0.0"
           args:
             exampleArg: "value"
    ...
   ```

4. `ContainerSpec`: container properties for the provider, consisting of:
   - Name (string): container name
   - ImageURL (optional string): container image URL
   - Args (optional map[string]string): extra provider specific flags
   - Env (optional []corev1.EnvVar): environment variables
   - Resources (optional corev1.ResourceRequirements): compute resources
   - Command (optional []string): override container's entrypoint array

   YAML example:

   ```yaml
   ...
   spec:
     deployment:
       containers:
         - name: "example-container"
           imageUrl: "example.com/repo/image-name:v1.0.0"
           args:
             exampleArg: "value"
           env:
             - name: "EXAMPLE_ENV"
               value: "example-value"
           resources:
             limits:
               cpu: "1"
               memory: "1Gi"
             requests:
               cpu: "500m"
               memory: "500Mi"
           command:
             - "/bin/bash"
   ...
   ```

5. `FetchConfiguration`: components and metadata fetch options, consisting of:
   - URL (optional string): URL for remote Github repository releases (e.g., "<https://github.com/owner/repo/releases>")
   - Selector (optional metav1.LabelSelector): label selector to use for fetching provider components and metadata from ConfigMaps stored in the cluster

   YAML example:

   ```yaml
   ...
   spec:
     fetchConfig:
       url: "https://github.com/owner/repo/releases"
       selector:
         matchLabels:
   ...
   ```

6. `SecretReference`: pointer to a secret object, consisting of:

- Name (string): name of the secret
- Namespace (optional string): namespace of the secret, defaults to the provider object namespace

  YAML example:

  ```yaml
  ...
  spec:
    configSecret:
      name: capa-secret
      namespace: capa-system
  ...
  ```
