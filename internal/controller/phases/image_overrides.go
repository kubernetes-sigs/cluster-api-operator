/*
Copyright 2024 The Kubernetes Authors.

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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
)

const (
	daemonSetKind = "DaemonSet"
)

func imageOverrides(component string, overrides configclient.Client) func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	imageOverridesWrapper := func(objs []unstructured.Unstructured) ([]unstructured.Unstructured, error) {
		if overrides == nil {
			return objs, nil
		}

		return fixImages(objs, func(image string) (string, error) {
			return overrides.ImageMeta().AlterImage(component, image)
		})
	}

	return imageOverridesWrapper
}

// fixImages alters images using the give alter func
// NB. The implemented approach is specific for the provider components YAML & for the cert-manager manifest; it is not
// intended to cover all the possible objects used to deploy containers existing in Kubernetes.
func fixImages(objs []unstructured.Unstructured, alterImageFunc func(image string) (string, error)) ([]unstructured.Unstructured, error) {
	for i := range objs {
		if err := fixDeploymentImages(&objs[i], alterImageFunc); err != nil {
			return nil, err
		}

		if err := fixDaemonSetImages(&objs[i], alterImageFunc); err != nil {
			return nil, err
		}
	}

	return objs, nil
}

func fixDeploymentImages(o *unstructured.Unstructured, alterImageFunc func(image string) (string, error)) error {
	if o.GetKind() != deploymentKind {
		return nil
	}

	// Convert Unstructured into a typed object
	d := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(o, d, nil); err != nil {
		return err
	}

	if err := fixPodSpecImages(&d.Spec.Template.Spec, alterImageFunc); err != nil {
		return fmt.Errorf("%w: failed to fix containers in deployment %s", err, d.Name)
	}

	// Convert typed object back to Unstructured
	return scheme.Scheme.Convert(d, o, nil)
}

func fixDaemonSetImages(o *unstructured.Unstructured, alterImageFunc func(image string) (string, error)) error {
	if o.GetKind() != daemonSetKind {
		return nil
	}

	// Convert Unstructured into a typed object
	d := &appsv1.DaemonSet{}
	if err := scheme.Scheme.Convert(o, d, nil); err != nil {
		return err
	}

	if err := fixPodSpecImages(&d.Spec.Template.Spec, alterImageFunc); err != nil {
		return fmt.Errorf("%w: failed to fix containers in deamonSet %s", err, d.Name)
	}
	// Convert typed object back to Unstructured
	return scheme.Scheme.Convert(d, o, nil)
}

func fixPodSpecImages(podSpec *corev1.PodSpec, alterImageFunc func(image string) (string, error)) error {
	if err := fixContainersImage(podSpec.Containers, alterImageFunc); err != nil {
		return fmt.Errorf("%w: failed to fix containers", err)
	}

	if err := fixContainersImage(podSpec.InitContainers, alterImageFunc); err != nil {
		return fmt.Errorf("%w: failed to fix init containers", err)
	}

	return nil
}

func fixContainersImage(containers []corev1.Container, alterImageFunc func(image string) (string, error)) error {
	for j := range containers {
		container := &containers[j]

		image, err := alterImageFunc(container.Image)
		if err != nil {
			return fmt.Errorf("%w: failed to fix image for container %s", err, container.Name)
		}

		container.Image = image
	}

	return nil
}
