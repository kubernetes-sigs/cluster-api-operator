/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	ProviderFinalizer         = "provider.cluster.x-k8s.io"
	ConfigMapVersionLabelName = "provider.cluster.x-k8s.io/version"
)

// ProviderSpec is the desired state of the Provider.
type ProviderSpec struct {
	// Version indicates the provider version.
	// +optional
	Version string `json:"version,omitempty"`

	// Manager defines the properties that can be enabled on the controller manager for the provider.
	// +optional
	Manager *ManagerSpec `json:"manager,omitempty"`

	// Deployment defines the properties that can be enabled on the deployment for the provider.
	// +optional
	Deployment *DeploymentSpec `json:"deployment,omitempty"`

	// SecretName is the name of the Secret providing the configuration
	// variables for the current provider instance, like e.g. credentials.
	// Such configurations will be used when creating or upgrading provider components.
	// The contents of the secret will be treated as immutable. If changes need
	// to be made, a new object can be created and the name should be updated.
	// The contents should be in the form of key:value. This secret must be in
	// the same namespace as the provider.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretNamespace is the namespace of the Secret providing the configuration variables. If not specified,
	// the namespace of the provider will be used.
	SecretNamespace string `json:"secretNamespace,omitempty"`

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
}

// ConfigmapReference contains enough information to locate the configmap.
type ConfigmapReference struct {
	// Name defines the name of the configmap.
	Name string `json:"name"`

	// Namespace defines the namespace of the configmap.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ManagerSpec defines the properties that can be enabled on the controller manager for the provider.
type ManagerSpec struct {
	// ControllerManager returns the configurations for controllers
	ControllerManager `json:",inline"`

	// ProfilerAddress defines the bind address to expose the pprof profiler (e.g. localhost:6060).
	// Default empty, meaning the profiler is disabled.
	// Controller Manager flag is --profiler-address.
	// +optional
	ProfilerAddress string `json:"profilerAddress,omitempty"`

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles
	// which can be run.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxConcurrentReconciles int `json:"maxConcurrentReconciles,omitempty"`

	// Verbosity set the logs verbosity. Defaults to 1.
	// Controller Manager flag is --verbosity.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Verbosity int `json:"verbosity,omitempty"`

	// FeatureGates define provider specific feature flags that will be passed
	// in as container args to the provider's controller manager.
	// Controller Manager flag is --feature-gates.
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
}

type ControllerManager struct {
	// Webhook contains the controllers webhook configuration
	// +optional
	Webhook ControllerWebhook `json:"webhook,omitempty"`

	// SyncPeriod determines the minimum frequency at which watched resources are
	// reconciled. A lower period will correct entropy more quickly, but reduce
	// responsiveness to change if there are many watched resources. Change this
	// value only if you know what you are doing. Defaults to 10 hours if unset.
	// there will a 10 percent jitter between the SyncPeriod of all controllers
	// so that all controllers will not send list requests simultaneously.
	// +optional
	SyncPeriod *metav1.Duration `json:"syncPeriod,omitempty"`

	// LeaderElection is the LeaderElection config to be used when configuring
	// the manager.Manager leader election
	// +optional
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty"`

	// Metrics contains thw controller metrics configuration
	// +optional
	Metrics ControllerMetrics `json:"metrics,omitempty"`

	// CacheNamespace if specified restricts the manager's cache to watch objects in
	// the desired namespace Defaults to all namespaces
	//
	// Note: If a namespace is specified, controllers can still Watch for a
	// cluster-scoped resource (e.g Node).  For namespaced resources the cache
	// will only hold objects from the desired namespace.
	// +optional
	CacheNamespace string `json:"cacheNamespace,omitempty"`

	// GracefulShutdownTimeout is the duration given to runnable to stop before the manager actually returns on stop.
	// To disable graceful shutdown, set to time.Duration(0)
	// To use graceful shutdown without timeout, set to a negative duration, e.G. time.Duration(-1)
	// The graceful shutdown is skipped for safety reasons in case the leader election lease is lost.
	GracefulShutdownTimeout *metav1.Duration `json:"gracefulShutDown,omitempty"`

	// Health contains the controller health configuration
	// +optional
	Health ControllerHealth `json:"health,omitempty"`

	// Controller contains global configuration options for controllers
	// registered within this manager.
	// +optional
	Controller *ControllerConfigurationSpec `json:"controller,omitempty"`
}

// ControllerWebhook defines the webhook server for the controller.
type ControllerWebhook struct {
	// Port is the port that the webhook server serves at.
	// It is used to set webhook.Server.Port.
	// +optional
	Port *int `json:"port,omitempty"`

	// Host is the hostname that the webhook server binds to.
	// It is used to set webhook.Server.Host.
	// +optional
	Host string `json:"host,omitempty"`

	// CertDir is the directory that contains the server key and certificate.
	// if not set, webhook server would look up the server key and certificate in
	// {TempDir}/k8s-webhook-server/serving-certs. The server key and certificate
	// must be named tls.key and tls.crt, respectively.
	// +optional
	CertDir string `json:"certDir,omitempty"`
}

// ControllerMetrics defines the metrics configs.
type ControllerMetrics struct {
	// BindAddress is the TCP address that the controller should bind to
	// for serving prometheus metrics.
	// It can be set to "0" to disable the metrics serving.
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`
}

// ControllerHealth defines the health configs.
type ControllerHealth struct {
	// HealthProbeBindAddress is the TCP address that the controller should bind to
	// for serving health probes
	// It can be set to "0" or "" to disable serving the health probe.
	// +optional
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`

	// ReadinessEndpointName, defaults to "readyz"
	// +optional
	ReadinessEndpointName string `json:"readinessEndpointName,omitempty"`

	// LivenessEndpointName, defaults to "healthz"
	// +optional
	LivenessEndpointName string `json:"livenessEndpointName,omitempty"`
}

// ControllerConfigurationSpec defines the global configuration for
// controllers registered with the manager.
type ControllerConfigurationSpec struct {
	// GroupKindConcurrency is a map from a Kind to the number of concurrent reconciliation
	// allowed for that controller.
	//
	// When a controller is registered within this manager using the builder utilities,
	// users have to specify the type the controller reconciles in the For(...) call.
	// If the object's kind passed matches one of the keys in this map, the concurrency
	// for that controller is set to the number specified.
	//
	// The key is expected to be consistent in form with GroupKind.String(),
	// e.g. ReplicaSet in apps group (regardless of version) would be `ReplicaSet.apps`.
	//
	// +optional
	GroupKindConcurrency map[string]int `json:"groupKindConcurrency,omitempty"`

	// CacheSyncTimeout refers to the time limit set to wait for syncing caches.
	// Defaults to 2 minutes if not set.
	// +optional
	CacheSyncTimeout *time.Duration `json:"cacheSyncTimeout,omitempty"`

	// RecoverPanic indicates if panics should be recovered.
	// +optional
	RecoverPanic *bool `json:"recoverPanic,omitempty"`
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
	Containers []ContainerSpec `json:"containers"`

	// If specified, the pod's service account
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// List of image pull secrets specified in the Deployment
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// ContainerSpec defines the properties available to override for each
// container in a provider deployment such as Image and Args to the container’s
// entrypoint.
type ContainerSpec struct {
	// Name of the container. Cannot be updated.
	Name string `json:"name"`

	// Container Image Name
	// +optional
	Image *ImageMeta `json:"image,omitempty"`

	// Args represents extra provider specific flags that are not encoded as fields in this API.
	// Explicit controller manager properties defined in the `Provider.ManagerSpec`
	// will have higher precedence than those defined in `ContainerSpec.Args`.
	// For example, `ManagerSpec.SyncPeriod` will be used instead of the
	// container arg `--sync-period` if both are defined.
	// The same holds for `ManagerSpec.FeatureGates` and `--feature-gates`.
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

// ImageMeta allows to customize the image used.
type ImageMeta struct {
	// Repository sets the container registry to pull images from.
	// +optional
	Repository string `json:"repository,omitempty"`

	// Name allows to specify a name for the image.
	// +optional
	Name string `json:"name,omitempty"`

	// Tag allows to specify a tag for the image.
	// +optional
	Tag string `json:"tag,omitempty"`
}

// FetchConfiguration determines the way to fetch the components and metadata for the provider.
type FetchConfiguration struct {
	// URL to be used for fetching the provider’s components and metadata from a remote Github repository.
	// For example, https://github.com/{owner}/{repository}/releases
	// You must set `providerSpec.Version` field for operator to pick up
	// desired version of the release from GitHub.
	// +optional
	URL string `json:"url,omitempty"`

	// Selector to be used for fetching provider’s components and metadata from
	// ConfigMaps stored inside the cluster. Each ConfigMap is expected to contain
	// components and metadata for a specific version only.
	// Note: the name of the ConfigMap should be set to the version or to override this
	// add a label like the following: provider.cluster.x-k8s.io/version=v1.4.3
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// ProviderStatus defines the observed state of the Provider.
type ProviderStatus struct {
	// Contract will contain the core provider contract that the provider is
	// abiding by, like e.g. v1alpha4.
	// +optional
	Contract *string `json:"contract,omitempty"`

	// Conditions define the current service state of the provider.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// InstalledVersion is the version of the provider that is installed.
	// +optional
	InstalledVersion *string `json:"installedVersion,omitempty"`
}
