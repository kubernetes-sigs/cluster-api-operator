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

package controller

import (
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubectl/pkg/cmd/util/podcmd"
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
)

const (
	deploymentKind       = "Deployment"
	namespaceKind        = "Namespace"
	managerContainerName = "manager"
)

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

			//nolint:nestif
			if o.GetKind() == deploymentKind {
				d := &appsv1.Deployment{}
				if err := scheme.Scheme.Convert(&o, d, nil); err != nil {
					return nil, err
				}

				providerDeployment := provider.GetSpec().Deployment

				// If there are multiple deployments, check if we specify customizations for those deployments.
				// We need to skip the deployment customization if there are several deployments available
				// and the deployment name doesn't follow "ca*-controller-manager" pattern, or the provider
				// doesn't specify customizations for the deployment.
				// This is a temporary fix until CAPI provides a contract to distinguish provider deployments.
				// TODO: replace this check and just compare labels when CAPI provides the contract for that.
				if isMultipleDeployments && !isProviderManagerDeploymentName(o.GetName()) {
					additionalDeployments := provider.GetSpec().AdditionalDeployments
					// Skip the deployment if there are no additional deployments specified.
					if additionalDeployments == nil {
						results = append(results, o)
						continue
					}

					additionalProviderCustomization, ok := additionalDeployments[o.GetName()]
					if !ok {
						// Skip if there is no customization for the deployment.
						results = append(results, o)
						continue
					}

					providerDeployment = &additionalProviderCustomization
				}

				if providerDeployment == nil {
					// Skip if there is no customization for the deployment.
					results = append(results, o)
					continue
				}

				customizeDeploymentSpec(*providerDeployment, d)

				if err := scheme.Scheme.Convert(d, &o, nil); err != nil {
					return nil, err
				}
			}

			results = append(results, o)
		}

		return results, nil
	}
}

func customizeDeploymentSpec(dSpec operatorv1.DeploymentSpec, d *appsv1.Deployment) {
	if dSpec.Replicas != nil {
		replicas := int32(*dSpec.Replicas) //nolint:gosec
		d.Spec.Replicas = ptr.To(replicas)
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

	// Apply container customizations.
	// Postpone customization of the container with empty name as it is treated differently.
	var managerContainer *operatorv1.ContainerSpec

	for _, pc := range dSpec.Containers {
		if pc.Name == "" {
			managerContainer = &pc
			continue
		}

		customizeContainer(pc, d)
	}

	if managerContainer != nil {
		managerContainer.Name = findManagerContainerName(&d.Spec)
		customizeContainer(*managerContainer, d)
	}
}

// findManagerContainer finds manager container in the provider deployment.
func findManagerContainerName(dSpec *appsv1.DeploymentSpec) string {
	// First check if the manager container name is specified in the provider spec.
	if dSpec.Template.Annotations[podcmd.DefaultContainerAnnotationName] != "" {
		return dSpec.Template.Annotations[podcmd.DefaultContainerAnnotationName]
	}

	// Then try to check if the manager container name is specified in the deployment spec.
	for ic := range dSpec.Template.Spec.Containers {
		if dSpec.Template.Spec.Containers[ic].Name == managerContainerName {
			return dSpec.Template.Spec.Containers[ic].Name
		}
	}

	// As a last resort, return the first container name.
	if len(dSpec.Template.Spec.Containers) > 0 {
		return dSpec.Template.Spec.Containers[0].Name
	}

	return ""
}

// customizeContainer customize provider container base on provider spec input.
func customizeContainer(cSpec operatorv1.ContainerSpec, d *appsv1.Deployment) {
	for j, c := range d.Spec.Template.Spec.Containers {
		if c.Name == cSpec.Name {
			// Sort the args map keys to ensure deterministic order
			argKeys := make([]string, 0, len(cSpec.Args))
			for k := range cSpec.Args {
				argKeys = append(argKeys, k)
			}

			sort.Strings(argKeys)

			// Process args in sorted order
			for _, an := range argKeys {
				c.Args = setArgs(c.Args, an, cSpec.Args[an])
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
		// Replace the argument if it already exists.
		if strings.HasPrefix(a, name+"=") {
			args[i] = name + "=" + value

			return args
		}
	}

	// Append the argument if it doesn't exist.
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
