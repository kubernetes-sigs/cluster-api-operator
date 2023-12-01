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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ GenericProvider = &AddonProvider{}

func (b *AddonProvider) GetConditions() clusterv1.Conditions {
	return b.Status.Conditions
}

func (b *AddonProvider) SetConditions(conditions clusterv1.Conditions) {
	b.Status.Conditions = conditions
}

func (b *AddonProvider) GetSpec() ProviderSpec {
	return b.Spec.ProviderSpec
}

func (b *AddonProvider) SetSpec(in ProviderSpec) {
	b.Spec.ProviderSpec = in
}

func (b *AddonProvider) GetStatus() ProviderStatus {
	return b.Status.ProviderStatus
}

func (b *AddonProvider) SetStatus(in ProviderStatus) {
	b.Status.ProviderStatus = in
}

func (b *AddonProvider) GetType() string {
	return "addon"
}

func (b *AddonProviderList) GetItems() []GenericProvider {
	providers := []GenericProvider{}

	for index := range b.Items {
		providers = append(providers, &b.Items[index])
	}

	return providers
}
