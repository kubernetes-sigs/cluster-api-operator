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

package v1alpha2

import (
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

const (
	DefaultManagerContainerName = ""
)

// Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec converts v1alpha2 ProviderSpec to v1alpha3.
func Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(in *ProviderSpec, out *operatorv1.ProviderSpec) error {
	out.Version = in.Version

	// Convert Deployment
	if in.Deployment != nil {
		out.Deployment = &operatorv1.DeploymentSpec{}
		if err := Convert_v1alpha2_DeploymentSpec_To_v1alpha3_DeploymentSpec(in.Deployment, out.Deployment); err != nil {
			return err
		}
	}

	// Merge Manager args into container args if Manager is specified
	if in.Manager != nil {
		managerArgs := convertManagerSpecToArgs(in.Manager)
		// Only create a synthetic manager container if we actually have args to persist
		if len(managerArgs) > 0 {
			// Create an empty Deployment spec if it doesn't exist
			if out.Deployment == nil {
				out.Deployment = &operatorv1.DeploymentSpec{}
			}

			// Create containers slice if it doesn't exist
			if out.Deployment.Containers == nil {
				out.Deployment.Containers = make([]operatorv1.ContainerSpec, 0)
			}

			container := &operatorv1.ContainerSpec{
				Name: DefaultManagerContainerName,
				Args: managerArgs,
			}

			// Insert at the end of the containers
			out.Deployment.Containers = append(out.Deployment.Containers, *container)
		}
	}

	// Copy other fields
	if in.ConfigSecret != nil {
		out.ConfigSecret = &operatorv1.SecretReference{
			Name:      in.ConfigSecret.Name,
			Namespace: in.ConfigSecret.Namespace,
		}
	}

	if in.FetchConfig != nil {
		out.FetchConfig = &operatorv1.FetchConfiguration{}
		if err := Convert_v1alpha2_FetchConfiguration_To_v1alpha3_FetchConfiguration(in.FetchConfig, out.FetchConfig); err != nil {
			return err
		}
	}

	if in.AdditionalManifestsRef != nil {
		out.AdditionalManifestsRef = &operatorv1.ConfigmapReference{
			Name:      in.AdditionalManifestsRef.Name,
			Namespace: in.AdditionalManifestsRef.Namespace,
		}
	}

	out.ManifestPatches = in.ManifestPatches

	// Convert AdditionalDeployments
	if in.AdditionalDeployments != nil {
		out.AdditionalDeployments = make(map[string]operatorv1.DeploymentSpec)

		for k, v := range in.AdditionalDeployments {
			out.AdditionalDeployments[k] = operatorv1.DeploymentSpec{}

			if v.Deployment != nil {
				deploymentSpec := operatorv1.DeploymentSpec{}
				if err := Convert_v1alpha2_DeploymentSpec_To_v1alpha3_DeploymentSpec(v.Deployment, &deploymentSpec); err != nil {
					return err
				}

				out.AdditionalDeployments[k] = deploymentSpec
			}
		}
	}

	return nil
}

// Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec converts v1alpha3 ProviderSpec to v1alpha2.
func Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(in *operatorv1.ProviderSpec, out *ProviderSpec) error {
	out.Version = in.Version

	// Convert Deployment
	if in.Deployment != nil {
		out.Deployment = &DeploymentSpec{}
		if err := Convert_v1alpha3_DeploymentSpec_To_v1alpha2_DeploymentSpec(in.Deployment, out.Deployment); err != nil {
			return err
		}

		// Extract Manager from container args
		if len(in.Deployment.Containers) > 0 {
			for i, container := range in.Deployment.Containers {
				if container.Name == DefaultManagerContainerName {
					if len(container.Args) > 0 {
						out.Manager = convertArgsToManagerSpec(container.Args)
					}

					// Remove manager container from the deployment
					out.Deployment.Containers = append(out.Deployment.Containers[:i], out.Deployment.Containers[i+1:]...)

					break
				}
			}
		}

		if len(out.Deployment.Containers) == 0 {
			out.Deployment.Containers = nil
		}

		// Prune empty deployment back to nil to preserve nil-vs-empty intent
		if IsV1alpha2DeploymentEmpty(out.Deployment) {
			out.Deployment = nil
		}
	}

	// Copy other fields
	if in.ConfigSecret != nil {
		out.ConfigSecret = &SecretReference{
			Name:      in.ConfigSecret.Name,
			Namespace: in.ConfigSecret.Namespace,
		}
	}

	if in.FetchConfig != nil {
		out.FetchConfig = &FetchConfiguration{}
		if err := Convert_v1alpha3_FetchConfiguration_To_v1alpha2_FetchConfiguration(in.FetchConfig, out.FetchConfig); err != nil {
			return err
		}
	}

	if in.AdditionalManifestsRef != nil {
		out.AdditionalManifestsRef = &ConfigmapReference{
			Name:      in.AdditionalManifestsRef.Name,
			Namespace: in.AdditionalManifestsRef.Namespace,
		}
	}

	out.ManifestPatches = in.ManifestPatches

	// Convert AdditionalDeployments
	if in.AdditionalDeployments != nil {
		out.AdditionalDeployments = make(map[string]AdditionalDeployments)

		for k, deploymentSpec := range in.AdditionalDeployments {
			ad := AdditionalDeployments{}

			ad.Deployment = &DeploymentSpec{}
			if err := Convert_v1alpha3_DeploymentSpec_To_v1alpha2_DeploymentSpec(&deploymentSpec, ad.Deployment); err != nil {
				return err
			}

			if IsV1alpha2DeploymentEmpty(ad.Deployment) {
				ad.Deployment = nil
			}

			out.AdditionalDeployments[k] = ad
		}
	}

	return nil
}

// Convert_v1alpha2_DeploymentSpec_To_v1alpha3_DeploymentSpec converts v1alpha2 DeploymentSpec to v1alpha3.
func Convert_v1alpha2_DeploymentSpec_To_v1alpha3_DeploymentSpec(in *DeploymentSpec, out *operatorv1.DeploymentSpec) error {
	out.Replicas = in.Replicas
	out.NodeSelector = in.NodeSelector
	out.Tolerations = in.Tolerations
	out.Affinity = in.Affinity

	if in.Containers != nil {
		out.Containers = make([]operatorv1.ContainerSpec, len(in.Containers))
		for i, c := range in.Containers {
			out.Containers[i] = operatorv1.ContainerSpec{
				Name:      c.Name,
				ImageURL:  c.ImageURL,
				Args:      c.Args,
				Env:       c.Env,
				Resources: c.Resources,
				Command:   c.Command,
			}
		}
	}

	if len(in.Containers) == 0 {
		out.Containers = nil
	}

	out.ServiceAccountName = in.ServiceAccountName
	out.ImagePullSecrets = in.ImagePullSecrets

	return nil
}

// Convert_v1alpha3_DeploymentSpec_To_v1alpha2_DeploymentSpec converts v1alpha3 DeploymentSpec to v1alpha2.
func Convert_v1alpha3_DeploymentSpec_To_v1alpha2_DeploymentSpec(in *operatorv1.DeploymentSpec, out *DeploymentSpec) error {
	out.Replicas = in.Replicas
	out.NodeSelector = in.NodeSelector
	out.Tolerations = in.Tolerations
	out.Affinity = in.Affinity

	if in.Containers != nil {
		out.Containers = make([]ContainerSpec, len(in.Containers))
		for i, c := range in.Containers {
			out.Containers[i] = ContainerSpec{
				Name:      c.Name,
				ImageURL:  c.ImageURL,
				Args:      c.Args,
				Env:       c.Env,
				Resources: c.Resources,
				Command:   c.Command,
			}
		}
	}

	if len(in.Containers) == 0 {
		out.Containers = nil
	}

	out.ServiceAccountName = in.ServiceAccountName
	out.ImagePullSecrets = in.ImagePullSecrets

	return nil
}

// Convert_v1alpha2_FetchConfiguration_To_v1alpha3_FetchConfiguration converts v1alpha2 FetchConfiguration to v1alpha3.
func Convert_v1alpha2_FetchConfiguration_To_v1alpha3_FetchConfiguration(in *FetchConfiguration, out *operatorv1.FetchConfiguration) error {
	out.OCI = in.OCI
	out.URL = in.URL
	out.Selector = in.Selector

	return nil
}

// Convert_v1alpha3_FetchConfiguration_To_v1alpha2_FetchConfiguration converts v1alpha3 FetchConfiguration to v1alpha2.
func Convert_v1alpha3_FetchConfiguration_To_v1alpha2_FetchConfiguration(in *operatorv1.FetchConfiguration, out *FetchConfiguration) error {
	out.OCI = in.OCI
	out.URL = in.URL
	out.Selector = in.Selector

	return nil
}

// IsV1alpha2DeploymentEmpty returns true if all fields in the deployment are zero values.
func IsV1alpha2DeploymentEmpty(d *DeploymentSpec) bool {
	if d == nil {
		return true
	}

	if d.Replicas != nil {
		return false
	}

	if len(d.NodeSelector) > 0 {
		return false
	}

	if len(d.Tolerations) > 0 {
		return false
	}

	if d.Affinity != nil {
		return false
	}

	if len(d.Containers) > 0 {
		return false
	}

	if d.ServiceAccountName != "" {
		return false
	}

	if len(d.ImagePullSecrets) > 0 {
		return false
	}

	return true
}
