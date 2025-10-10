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

// IsV1alpha3DeploymentEmpty returns true if all fields in the deployment are zero values.
func IsV1alpha3DeploymentEmpty(d *DeploymentSpec) bool {
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
