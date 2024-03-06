# Provider List

The Cluster API Operator introduces new API types: `CoreProvider`, `BootstrapProvider`, `ControlPlaneProvider`, `InfrastructureProvider`, `AddonProvider` and `IPAMProvider`. These five provider types share common Spec and Status types, `ProviderSpec` and `ProviderStatus`, respectively.

The CRDs are scoped to be namespaced, allowing RBAC restrictions to be enforced if needed. This scoping also enables the installation of multiple versions of controllers (grouped within namespaces) in the same management cluster.

Related Golang structs can be found in the [Cluster API Operator repository](https://github.com/kubernetes-sigs/cluster-api-operator/tree/main/api/v1alpha1).

Below are the new API types being defined, with shared types used for Spec and Status among the different provider types—Core, Bootstrap, ControlPlane, and Infrastructure:

*CoreProvider*

```golang
type CoreProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*BootstrapProvider*

```golang
type BootstrapProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*ControlPlaneProvider*

```golang
type ControlPlaneProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*InfrastructureProvider*

```golang
type InfrastructureProvider struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   ProviderSpec   `json:"spec,omitempty"`
  Status ProviderStatus `json:"status,omitempty"`
}
```

*AddonProvider*

```golang
type AddonProvider struct {
 metav1.TypeMeta   `json:",inline"`
 metav1.ObjectMeta `json:"metadata,omitempty"`

 Spec   AddonProviderSpec   `json:"spec,omitempty"`
 Status AddonProviderStatus `json:"status,omitempty"`
}
```

*IPAMProvider*

```golang
type IPAMProvider struct {
 metav1.TypeMeta   `json:",inline"`
 metav1.ObjectMeta `json:"metadata,omitempty"`

 Spec   IPAMProviderSpec   `json:"spec,omitempty"`
 Status IPAMProviderStatus `json:"status,omitempty"`
}
```

The following sections provide details about `ProviderSpec` and `ProviderStatus`, which are shared among all the provider types.

## Provider Status

`ProviderStatus`: observed state of the Provider, consisting of:

- Contract (optional string): core provider contract being adhered to (e.g., "v1beta1")
- Conditions (optional clusterv1.Conditions): current service state of the provider
- ObservedGeneration (optional int64): latest generation observed by the controller
- InstalledVersion (optional string): version of the provider that is installed

   YAML example:

   ```yaml
   status:
     contract: "v1beta1"
     conditions:
       - type: "Ready"
         status: "True"
         reason: "ProviderAvailable"
         message: "Provider is available and ready"
     observedGeneration: 1
     installedVersion: "v0.1.0"
   ```
