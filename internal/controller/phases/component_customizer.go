/*
Copyright 2022 The Kubernetes Authors.

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

package phases

import (
	"fmt"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/utils/pointer"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
)

const (
	deploymentKind       = "Deployment"
	namespaceKind        = "Namespace"
	managerContainerName = "manager"
	defaultVerbosity     = 1
)

var bool2Str = map[bool]string{true: "true", false: "false"}

// customizeObjectsFn apply provider specific customization to a list of manifests.
func customizeObjectsFn(provider operatorv1.GenericProvider) func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	return func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
		results := []unstructured.Unstructured{}

		isMultipleDeployments := isMultipleDeployments(objs)

		for i := range objs {
			o := objs[i]

			if o.GetKind() == namespaceKind {
				// filter out namespaces as the targetNamespace already exists as the provider object is in it.
				continue
			}

			if o.GetNamespace() != "" {
				// only set the ownership on namespaced objects.
				ownerReferences := o.GetOwnerReferences()
				if ownerReferences == nil {
					ownerReferences = []metav1.OwnerReference{}
				}

				o.SetOwnerReferences(util.EnsureOwnerRef(ownerReferences,
					metav1.OwnerReference{
						APIVersion: operatorv1.GroupVersion.String(),
						Kind:       provider.GetObjectKind().GroupVersionKind().Kind,
						Name:       provider.GetName(),
						UID:        provider.GetUID(),
					}))
			}

			if o.GetKind() == deploymentKind {
				// We need to skip the deployment customization if there are several deployments available
				// and the deployment name doesn't follow "ca*-controller-manager" pattern.
				// Currently it is applicable only for CAPZ manifests, which contain 2 deployments:
				// capz-controller-manager and azureserviceoperator-controller-manager
				// This is a temporary fix until CAPI provides a contract to distinguish provider deployments.
				// TODO: replace this check and just compare labels when CAPI provides the contract for that.
				if isMultipleDeployments && !isProviderManagerDeploymentName(o.GetName()) {
					results = append(results, o)

					continue
				}

				d := &appsv1.Deployment{}
				if err := scheme.Scheme.Convert(&o, d, nil); err != nil {
					return nil, err
				}

				if err := customizeDeployment(provider.GetSpec(), d); err != nil {
					return nil, err
				}

				if err := scheme.Scheme.Convert(d, &o, nil); err != nil {
					return nil, err
				}
			}

			results = append(results, o)
		}

		return results, nil
	}
}

// customizeDeployment customize provider deployment base on provider spec input.
func customizeDeployment(pSpec operatorv1.ProviderSpec, d *appsv1.Deployment) error {
	// Customize deployment spec first.
	if pSpec.Deployment != nil {
		customizeDeploymentSpec(pSpec, d)
	}

	// Run the customizeManagerContainer after so it overrides anything in the deploymentSpec.
	if pSpec.Manager != nil {
		container := findManagerContainer(&d.Spec)
		if container == nil {
			return fmt.Errorf("cannot find %q container in deployment %q", managerContainerName, d.Name)
		}

		customizeManagerContainer(pSpec.Manager, container)
	}

	return nil
}

func customizeDeploymentSpec(pSpec operatorv1.ProviderSpec, d *appsv1.Deployment) {
	dSpec := pSpec.Deployment

	if dSpec.Replicas != nil {
		d.Spec.Replicas = pointer.Int32(int32(*dSpec.Replicas))
	}

	if dSpec.Affinity != nil {
		d.Spec.Template.Spec.Affinity = dSpec.Affinity
	}

	if dSpec.NodeSelector != nil {
		d.Spec.Template.Spec.NodeSelector = dSpec.NodeSelector
	}

	if dSpec.Tolerations != nil {
		d.Spec.Template.Spec.Tolerations = dSpec.Tolerations
	}

	if dSpec.ServiceAccountName != "" {
		d.Spec.Template.Spec.ServiceAccountName = dSpec.ServiceAccountName
	}

	if dSpec.ImagePullSecrets != nil {
		d.Spec.Template.Spec.ImagePullSecrets = dSpec.ImagePullSecrets
	}

	for _, pc := range dSpec.Containers {
		customizeContainer(pc, d)
	}
}

// findManagerContainer finds manager container in the provider deployment.
func findManagerContainer(dSpec *appsv1.DeploymentSpec) *corev1.Container {
	for ic := range dSpec.Template.Spec.Containers {
		return &dSpec.Template.Spec.Containers[ic]
	}

	return nil
}

// customizeManagerContainer customize manager container base on provider spec input.
func customizeManagerContainer(mSpec *operatorv1.ManagerSpec, c *corev1.Container) {
	// ControllerManagerConfigurationSpec fields
	if mSpec.Controller != nil {
		// TODO can't find an arg for CacheSyncTimeout
		for k, v := range mSpec.Controller.GroupKindConcurrency {
			c.Args = setArgs(c.Args, "--"+strings.ToLower(k)+"-concurrency", fmt.Sprint(v))
		}
	}

	if mSpec.MaxConcurrentReconciles != 0 {
		c.Args = setArgs(c.Args, "--max-concurrent-reconciles", fmt.Sprint(mSpec.MaxConcurrentReconciles))
	}

	if mSpec.CacheNamespace != "" {
		// This field seems somewhat in conflict with:
		// The `ContainerSpec.Args` will ignore the key `namespace` since the operator
		// enforces a deployment model where all the providers should be configured to
		// watch all the namespaces.
		c.Args = setArgs(c.Args, "--namespace", mSpec.CacheNamespace)
	}

	// TODO can't find an arg for GracefulShutdownTimeout

	if mSpec.Health.HealthProbeBindAddress != "" {
		c.Args = setArgs(c.Args, "--health-addr", mSpec.Health.HealthProbeBindAddress)
	}

	if mSpec.Health.LivenessEndpointName != "" && c.LivenessProbe != nil && c.LivenessProbe.HTTPGet != nil {
		c.LivenessProbe.HTTPGet.Path = "/" + mSpec.Health.LivenessEndpointName
	}

	if mSpec.Health.ReadinessEndpointName != "" && c.ReadinessProbe != nil && c.ReadinessProbe.HTTPGet != nil {
		c.ReadinessProbe.HTTPGet.Path = "/" + mSpec.Health.ReadinessEndpointName
	}

	if mSpec.LeaderElection != nil && mSpec.LeaderElection.LeaderElect != nil {
		c.Args = leaderElectionArgs(mSpec.LeaderElection, c.Args)
	}

	if mSpec.Metrics.BindAddress != "" {
		c.Args = setArgs(c.Args, "--metrics-bind-addr", mSpec.Metrics.BindAddress)
	}

	// webhooks
	if mSpec.Webhook.Host != "" {
		c.Args = setArgs(c.Args, "--webhook-host", mSpec.Webhook.Host)
	}

	if mSpec.Webhook.Port != nil {
		c.Args = setArgs(c.Args, "--webhook-port", fmt.Sprint(*mSpec.Webhook.Port))
	}

	if mSpec.Webhook.CertDir != "" {
		c.Args = setArgs(c.Args, "--webhook-cert-dir", mSpec.Webhook.CertDir)
	}

	// top level fields
	if mSpec.SyncPeriod != nil {
		syncPeriod := int(mSpec.SyncPeriod.Duration.Round(time.Second).Seconds())
		if syncPeriod > 0 {
			c.Args = setArgs(c.Args, "--sync-period", fmt.Sprintf("%ds", syncPeriod))
		}
	}

	if mSpec.ProfilerAddress != "" {
		c.Args = setArgs(c.Args, "--profiler-address", mSpec.ProfilerAddress)
	}

	if mSpec.Verbosity != defaultVerbosity {
		c.Args = setArgs(c.Args, "--v", fmt.Sprint(mSpec.Verbosity))
	}

	if len(mSpec.FeatureGates) > 0 {
		fgValue := []string{}

		for fg, val := range mSpec.FeatureGates {
			fgValue = append(fgValue, fg+"="+bool2Str[val])
		}

		sort.Strings(fgValue)
		c.Args = setArgs(c.Args, "--feature-gates", strings.Join(fgValue, ","))
	}
}

// customizeContainer customize provider container base on provider spec input.
func customizeContainer(cSpec operatorv1.ContainerSpec, d *appsv1.Deployment) {
	for j, c := range d.Spec.Template.Spec.Containers {
		if c.Name == cSpec.Name {
			for an, av := range cSpec.Args {
				// The `ContainerSpec.Args` will ignore the key `namespace` since the operator
				// enforces a deployment model where all the providers should be configured to
				// watch all the namespaces.
				if an != "namespace" {
					c.Args = setArgs(c.Args, an, av)
				}
			}

			for _, se := range cSpec.Env {
				c.Env = removeEnv(c.Env, se.Name)
				c.Env = append(c.Env, se)
			}

			if cSpec.Resources != nil {
				c.Resources = *cSpec.Resources
			}

			if cSpec.ImageURL != nil {
				c.Image = *cSpec.ImageURL
			}

			if cSpec.Command != nil {
				c.Command = cSpec.Command
			}
		}

		d.Spec.Template.Spec.Containers[j] = c
	}
}

// setArg set container arguments.
func setArgs(args []string, name, value string) []string {
	for i, a := range args {
		if strings.HasPrefix(a, name+"=") {
			args[i] = name + "=" + value

			return args
		}
	}

	return append(args, name+"="+value)
}

// removeEnv remove container environment.
func removeEnv(envs []corev1.EnvVar, name string) []corev1.EnvVar {
	for i, a := range envs {
		if a.Name == name {
			copy(envs[i:], envs[i+1:])

			return envs[:len(envs)-1]
		}
	}

	return envs
}

// leaderElectionArgs set leader election flags.
func leaderElectionArgs(lec *configv1alpha1.LeaderElectionConfiguration, args []string) []string {
	args = setArgs(args, "--leader-elect", bool2Str[*lec.LeaderElect])

	if *lec.LeaderElect {
		if lec.ResourceName != "" && lec.ResourceNamespace != "" {
			args = setArgs(args, "--leader-election-id", lec.ResourceNamespace+"/"+lec.ResourceName)
		}

		leaseDuration := int(lec.LeaseDuration.Duration.Round(time.Second).Seconds())

		if leaseDuration > 0 {
			args = setArgs(args, "--leader-elect-lease-duration", fmt.Sprintf("%ds", leaseDuration))
		}

		renewDuration := int(lec.RenewDeadline.Duration.Round(time.Second).Seconds())

		if renewDuration > 0 {
			args = setArgs(args, "--leader-elect-renew-deadline", fmt.Sprintf("%ds", renewDuration))
		}

		retryDuration := int(lec.RetryPeriod.Duration.Round(time.Second).Seconds())

		if retryDuration > 0 {
			args = setArgs(args, "--leader-elect-retry-period", fmt.Sprintf("%ds", retryDuration))
		}
	}

	return args
}

// isMultipleDeployments check if there are multiple deployments in the manifests.
func isMultipleDeployments(objs []unstructured.Unstructured) bool {
	var numberOfDeployments int

	for i := range objs {
		o := objs[i]

		if o.GetKind() == deploymentKind {
			numberOfDeployments++
		}
	}

	return numberOfDeployments > 1
}

// isProviderManagerDeploymentName checks that the provided follows the provider manager deployment name pattern: "ca*-controller-manager".
func isProviderManagerDeploymentName(name string) bool {
	return strings.HasPrefix(name, "ca") && strings.HasSuffix(name, "-controller-manager")
}
