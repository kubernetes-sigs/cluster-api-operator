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

package v1alpha2

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ GenericProvider = &RuntimeExtensionProvider{}

func (p *RuntimeExtensionProvider) GetConditions() clusterv1.Conditions {
	return p.Status.Conditions
}

func (p *RuntimeExtensionProvider) SetConditions(conditions clusterv1.Conditions) {
	p.Status.Conditions = conditions
}

func (p *RuntimeExtensionProvider) GetSpec() ProviderSpec {
	return p.Spec.ProviderSpec
}

func (p *RuntimeExtensionProvider) SetSpec(in ProviderSpec) {
	p.Spec.ProviderSpec = in
}

func (p *RuntimeExtensionProvider) GetStatus() ProviderStatus {
	return p.Status.ProviderStatus
}

func (p *RuntimeExtensionProvider) SetStatus(in ProviderStatus) {
	p.Status.ProviderStatus = in
}

func (p *RuntimeExtensionProvider) GetType() string {
	return "runtimeextension"
}

func (p *RuntimeExtensionProvider) ProviderName() string {
	return p.GetName()
}

func (p *RuntimeExtensionProviderList) GetItems() []GenericProvider {
	providers := []GenericProvider{}

	for index := range p.Items {
		providers = append(providers, &p.Items[index])
	}

	return providers
}
