/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProviderFinalizer = "provider.cluster.x-k8s.io"

	ConfigMapVersionLabelName = "provider.cluster.x-k8s.io/version"
	ConfigMapTypeLabel        = "provider.cluster.x-k8s.io/type"
	ConfigMapNameLabel        = "provider.cluster.x-k8s.io/name"

	CompressedAnnotation = "provider.cluster.x-k8s.io/compressed"
	TrueValue            = "true"

	MetadataConfigMapKey            = "metadata"
	ComponentsConfigMapKey          = "components"
	AdditionalManifestsConfigMapKey = "manifests"
)

// ProviderSpec is the desired state of the Provider.
type ProviderSpec struct {
	// Version indicates the provider version.
	// +optional
	Version string `json:"version,omitempty"`

	// Deployment defines the properties that can be enabled on the deployment for the provider.
	// +optional
	Deployment *DeploymentSpec `json:"deployment,omitempty"`

	// ConfigSecret is the object with name and namespace of the Secret providing
	// the configuration variables for the current provider instance, like e.g. credentials.
	// Such configurations will be used when creating or upgrading provider components.
	// The contents of the secret will be treated as immutable. If changes need
	// to be made, a new object can be created and the name should be updated.
	// The contents should be in the form of key:value. This secret must be in
	// the same namespace as the provider.
	// +optional
	ConfigSecret *SecretReference `json:"configSecret,omitempty"`

	// FetchConfig determines how the operator will fetch the components and metadata for the provider.
	// If nil, the operator will try to fetch components according to default
	// embedded fetch configuration for the given kind and `ObjectMeta.Name`.
	// For example, the infrastructure name `aws` will fetch artifacts from
	// https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases.
	// +optional
	FetchConfig *FetchConfiguration `json:"fetchConfig,omitempty"`

	// AdditionalManifests is reference to configmap that contains additional manifests that will be applied
	// together with the provider components. The key for storing these manifests has to be `manifests`.
	// The manifests are applied only once when a certain release is installed/upgraded. If namespace is not specified, the
	// namespace of the provider will be used. There is no validation of the yaml content inside the configmap.
	// +optional
	AdditionalManifestsRef *ConfigmapReference `json:"additionalManifests,omitempty"`

	// ManifestPatches are applied to rendered provider manifests to customize the
	// provider manifests. Patches are applied in the order they are specified.
	// The `kind` field must match the target object, and
	// if `apiVersion` is specified it will only be applied to matching objects.
	// This should be an inline yaml blob-string https://datatracker.ietf.org/doc/html/rfc7396
	// +optional
	ManifestPatches []string `json:"manifestPatches,omitempty"`

	// AdditionalDeployments is a map of additional deployments that the provider
	// should manage. The key is the name of the deployment and the value is the
	// DeploymentSpec.
	// +optional
	AdditionalDeployments map[string]DeploymentSpec `json:"additionalDeployments,omitempty"`
}

// ConfigmapReference contains enough information to locate the configmap.
type ConfigmapReference struct {
	// Name defines the name of the configmap.
	Name string `json:"name"`

	// Namespace defines the namespace of the configmap.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// SecretReference contains enough information to locate the referenced secret.
type SecretReference struct {
	// Name defines the name of the secret.
	Name string `json:"name"`

	// Namespace defines the namespace of the secret.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// DeploymentSpec defines the properties that can be enabled on the Deployment for the provider.
type DeploymentSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit zero and not specified. Defaults to 1.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Replicas *int `json:"replicas,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// List of containers specified in the Deployment
	// +optional
	Containers []ContainerSpec `json:"containers,omitempty"`

	// If specified, the pod's service account
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// List of image pull secrets specified in the Deployment
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// ContainerSpec defines the properties available to override for each
// container in a provider deployment such as Image and Args to the container's
// entrypoint.
type ContainerSpec struct {
	// Name of the container. Cannot be updated.
	Name string `json:"name"`

	// Container Image URL
	// +optional
	ImageURL *string `json:"imageUrl,omitempty"`

	// Args represents provider specific flags that can be passed directly to the container's entrypoint.
	// This includes both additional flags and those that were previously managed through ManagerSpec.
	//
	// Common arguments that replace ManagerSpec fields:
	// - Controller concurrency: --<resource>-concurrency (e.g., --cluster-concurrency=10)
	// - Max concurrent reconciles: --max-concurrent-reconciles=<number>
	// - Cache namespace: --namespace=<namespace>
	// - Health probe: --health-addr=<address>
	// - Leader election: --leader-elect=<true/false>, --leader-election-id=<id>
	// - Leader election durations: --leader-elect-lease-duration=<duration>s,
	//   --leader-elect-renew-deadline=<duration>s, --leader-elect-retry-period=<duration>s
	// - Metrics: --metrics-bind-addr=<address>
	// - Webhook: --webhook-host=<host>, --webhook-port=<port>, --webhook-cert-dir=<dir>
	// - Sync period: --sync-period=<duration>s
	// - Profiler: --profiler-address=<address>
	// - Verbosity: --v=<level>
	// - Feature gates: --feature-gates=<gate1>=<true/false>,<gate2>=<true/false>
	//
	// Note: Some fields like LivenessEndpointName and ReadinessEndpointName should be
	// configured by modifying the container's probe paths directly.
	// +optional
	Args map[string]string `json:"args,omitempty"`

	// List of environment variables to set in the container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute resources required by this container.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Command allows override container's entrypoint array.
	Command []string `json:"command,omitempty"`
}

// FetchConfiguration determines the way to fetch the components and metadata for the provider.
// +kubebuilder:validation:XValidation:rule="[has(self.oci), has(self.url), has(self.selector)].exists_one(x,x)", message="Must specify one and only one of {oci, url, selector}"
type FetchConfiguration struct {
	// OCI configurations to be used for fetching the provider's components and metadata from an OCI artifact.
	OCIConfiguration `json:",inline"`

	// URL to be used for fetching the provider's components and metadata from a remote Github repository.
	// For example, https://github.com/{owner}/{repository}/releases
	// You must set `providerSpec.Version` field for operator to pick up
	// desired version of the release from GitHub.
	// +optional
	URL string `json:"url,omitempty"`

	// Selector to be used for fetching provider's components and metadata from
	// ConfigMaps stored inside the cluster. Each ConfigMap is expected to contain
	// components and metadata for a specific version only.
	// Note: the name of the ConfigMap should be set to the version or to override this
	// add a label like the following: provider.cluster.x-k8s.io/version=v1.4.3
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type OCIConfiguration struct {
	// OCI to be used for fetching the provider's components and metadata from an OCI artifact.
	// You must set `providerSpec.Version` field for operator to pick up desired version of the release from GitHub.
	// If the providerSpec.Version is missing, latest provider version from clusterctl defaults is used.
	// +optional
	OCI string `json:"oci,omitempty"`
}

// ProviderStatus defines the observed state of the Provider.
type ProviderStatus struct {
	// Contract will contain the core provider contract that the provider is
	// abiding by, like e.g. v1alpha4.
	// +optional
	Contract *string `json:"contract,omitempty"`

	// Conditions define the current service state of the provider.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// InstalledVersion is the version of the provider that is installed.
	// +optional
	InstalledVersion *string `json:"installedVersion,omitempty"`
}
