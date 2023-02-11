/*
Copyright 2021 The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// CoreProviderSpec defines the desired state of CoreProvider.
type CoreProviderSpec struct {
	ProviderSpec `json:",inline"`
}

// CoreProviderStatus defines the observed state of CoreProvider.
type CoreProviderStatus struct {
	ProviderStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="InstalledVersion",type="string",JSONPath=".status.installedVersion"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"

// CoreProvider is the Schema for the coreproviders API.
type CoreProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoreProviderSpec   `json:"spec,omitempty"`
	Status CoreProviderStatus `json:"status,omitempty"`
}

func (b *CoreProvider) GetConditions() clusterv1.Conditions {
	return b.Status.Conditions
}

func (b *CoreProvider) SetConditions(conditions clusterv1.Conditions) {
	b.Status.Conditions = conditions
}

func (b *CoreProvider) GetSpec() ProviderSpec {
	return b.Spec.ProviderSpec
}

func (b *CoreProvider) SetSpec(in ProviderSpec) {
	b.Spec.ProviderSpec = in
}

func (b *CoreProvider) GetStatus() ProviderStatus {
	return b.Status.ProviderStatus
}

func (b *CoreProvider) SetStatus(in ProviderStatus) {
	b.Status.ProviderStatus = in
}

// +kubebuilder:object:root=true

// CoreProviderList contains a list of CoreProvider.
type CoreProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoreProvider `json:"items"`
}

func (c *CoreProviderList) GetItems() []GenericProvider {
	providers := []GenericProvider{}
	for _, provider := range c.Items {
		providers = append(providers, &provider)
	}

	return providers
}

func init() {
	SchemeBuilder.Register(&CoreProvider{}, &CoreProviderList{})
}
