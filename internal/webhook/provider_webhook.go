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

package webhook

import (
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

// setDefaultProviderSpec sets the default values for the provider spec.
func setDefaultProviderSpec(providerSpec *operatorv1.ProviderSpec, providerNamespace string) {
	if providerSpec.ConfigSecret != nil && providerSpec.ConfigSecret.Namespace == "" {
		providerSpec.ConfigSecret.Namespace = providerNamespace
	}

	if providerSpec.AdditionalManifestsRef != nil && providerSpec.AdditionalManifestsRef.Namespace == "" {
		providerSpec.AdditionalManifestsRef.Namespace = providerNamespace
	}
}
