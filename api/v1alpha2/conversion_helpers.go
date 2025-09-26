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
	"strings"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

const (
	managerContainerName = "manager"
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
	if in.Manager != nil && out.Deployment != nil && len(out.Deployment.Containers) > 0 {
		managerArgs := convertManagerSpecToArgs(in.Manager)

		// Find the manager container (usually the first one or one with name "manager")
		for i := range out.Deployment.Containers {
			container := &out.Deployment.Containers[i]
			if container.Name == managerContainerName || i == 0 {
				// Merge manager args with existing container args
				if container.Args == nil {
					container.Args = make(map[string]string)
				}

				for k, v := range managerArgs {
					// Manager args take precedence over container args
					container.Args[k] = v
				}

				break
			}
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
			if v.Deployment != nil {
				deploymentSpec := operatorv1.DeploymentSpec{}
				if err := Convert_v1alpha2_DeploymentSpec_To_v1alpha3_DeploymentSpec(v.Deployment, &deploymentSpec); err != nil {
					return err
				}

				// Merge Manager args into container args if Manager is specified
				if v.Manager != nil && len(deploymentSpec.Containers) > 0 {
					managerArgs := convertManagerSpecToArgs(v.Manager)

					for i := range deploymentSpec.Containers {
						container := &deploymentSpec.Containers[i]
						if container.Name == managerContainerName || i == 0 {
							if container.Args == nil {
								container.Args = make(map[string]string)
							}

							for k, v := range managerArgs {
								container.Args[k] = v
							}

							break
						}
					}
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
			for _, container := range in.Deployment.Containers {
				if container.Name == managerContainerName || len(in.Deployment.Containers) == 1 {
					if len(container.Args) > 0 {
						out.Manager = convertArgsToManagerSpec(container.Args)

						// Remove manager-related args from container args in v1alpha2
						if out.Deployment != nil && len(out.Deployment.Containers) > 0 {
							for i := range out.Deployment.Containers {
								if out.Deployment.Containers[i].Name == container.Name {
									// Create new args map without manager-specific args
									// Note: Only known manager args are removed, other args remain
									newArgs := make(map[string]string)

									for k, v := range container.Args {
										if !isManagerArg(k) {
											newArgs[k] = v
										}
									}

									if len(newArgs) > 0 {
										out.Deployment.Containers[i].Args = newArgs
									} else {
										out.Deployment.Containers[i].Args = nil
									}

									break
								}
							}
						}
					}

					break
				}
			}
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

			// Extract Manager from container args
			if len(deploymentSpec.Containers) > 0 {
				for _, container := range deploymentSpec.Containers {
					if container.Name == managerContainerName || len(deploymentSpec.Containers) == 1 {
						if len(container.Args) > 0 {
							ad.Manager = convertArgsToManagerSpec(container.Args)

							// Remove manager-related args from container args in v1alpha2
							if ad.Deployment != nil && len(ad.Deployment.Containers) > 0 {
								for i := range ad.Deployment.Containers {
									if ad.Deployment.Containers[i].Name == container.Name {
										// Create new args map without manager-specific args
										// Note: Only known manager args are removed, other args remain
										newArgs := make(map[string]string)

										for k, v := range container.Args {
											if !isManagerArg(k) {
												newArgs[k] = v
											}
										}

										if len(newArgs) > 0 {
											ad.Deployment.Containers[i].Args = newArgs
										} else {
											ad.Deployment.Containers[i].Args = nil
										}

										break
									}
								}
							}
						}

						break
					}
				}
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

// isManagerArg checks if an argument is a manager-specific argument.
func isManagerArg(arg string) bool {
	managerArgs := []string{
		"--max-concurrent-reconciles",
		"--namespace",
		"--health-addr",
		"--leader-elect",
		"--leader-election-id",
		"--leader-elect-lease-duration",
		"--leader-elect-renew-deadline",
		"--leader-elect-retry-period",
		"--metrics-bind-addr",
		"--webhook-host",
		"--webhook-port",
		"--webhook-cert-dir",
		"--sync-period",
		"--profiler-address",
		"--v",
		"--feature-gates",
	}

	for _, ma := range managerArgs {
		if arg == ma {
			return true
		}
	}

	// Also check for concurrency args
	if strings.HasSuffix(arg, "-concurrency") {
		return true
	}

	return false
}
