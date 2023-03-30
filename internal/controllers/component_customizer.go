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

package controllers

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
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api-operator/internal/controllers/genericprovider"
	"sigs.k8s.io/cluster-api/util"
)

const (
	deploymentKind       = "Deployment"
	namespaceKind        = "Namespace"
	managerContainerName = "manager"
	defaultVerbosity     = 1
)

var (
	bool2Str = map[bool]string{true: "true", false: "false"}
)

// customizeObjectsFn apply provider specific customization to a list of manifests.
func customizeObjectsFn(provider genericprovider.GenericProvider) func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	return func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
		results := []unstructured.Unstructured{}

		for i := range objs {
			o := objs[i]

			if o.GetKind() == namespaceKind {
				// filter out namespaces as the targetNamespace already exists as the provider object is in it.
				continue
			}

			if o.GetNamespace() != "" {
				// only set the ownership on namespaced objects.
				o.SetOwnerReferences(util.EnsureOwnerRef(provider.GetOwnerReferences(),
					metav1.OwnerReference{
						APIVersion: operatorv1.GroupVersion.String(),
						Kind:       provider.GetObjectKind().GroupVersionKind().Kind,
						Name:       provider.GetName(),
						UID:        provider.GetUID(),
					}))
			}

			if o.GetKind() == deploymentKind {
				d := &appsv1.Deployment{}
				if err := scheme.Scheme.Convert(&o, d, nil); err != nil {
					return nil, err
				}

				if provider.GetSpec().Deployment != nil {
					customizeDeployment(provider.GetSpec(), d)
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
func customizeDeployment(pSpec operatorv1.ProviderSpec, d *appsv1.Deployment) {
	if pSpec.Deployment != nil {
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

		for _, pc := range dSpec.Containers {
			customizeContainer(pc, d)
		}
	}

	// run the customizeManager last so it overrides anything in the deploymentSpec.
	if pSpec.Manager != nil {
		for ic, c := range d.Spec.Template.Spec.Containers {
			if c.Name == managerContainerName {
				customizeManager(pSpec.Manager, &d.Spec.Template.Spec.Containers[ic])
			}
		}
	}
}

// customizeManager customize manager container base on provider spec input.
func customizeManager(mSpec *operatorv1.ManagerSpec, c *corev1.Container) {
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
		// This field seems somewhat in confilict with:
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
		// TODO or --metrics-bind-addr
		c.Args = setArgs(c.Args, "--metrics-addr", mSpec.Metrics.BindAddress)
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

			if cSpec.Image != nil && cSpec.Image.Name != "" && cSpec.Image.Repository != "" {
				c.Image = imageMetaToURL(cSpec.Image)
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

// imageMetaToURL translate container image meta to URL.
func imageMetaToURL(im *operatorv1.ImageMeta) string {
	tag := "latest"
	if im.Tag != "" {
		tag = im.Tag
	}

	return strings.Join([]string{im.Repository, im.Name}, "/") + ":" + tag
}

// leaderElectionArgs set leader election flags.
func leaderElectionArgs(lec *configv1alpha1.LeaderElectionConfiguration, args []string) []string {
	args = setArgs(args, "--enable-leader-election", bool2Str[*lec.LeaderElect])

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
