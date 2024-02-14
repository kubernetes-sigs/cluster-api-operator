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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IPAMProviderSpec defines the desired state of IPAMProvider.
type IPAMProviderSpec struct {
	ProviderSpec `json:",inline"`
}

// IPAMProviderStatus defines the observed state of IPAMProvider.
type IPAMProviderStatus struct {
	ProviderStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ipamproviders,shortName=caipamp,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="InstalledVersion",type="string",JSONPath=".status.installedVersion"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:storageversion

// IPAMProvider is the Schema for the IPAMProviders API.
type IPAMProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAMProviderSpec   `json:"spec,omitempty"`
	Status IPAMProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPAMProviderList contains a list of IPAMProvider.
type IPAMProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAMProvider `json:"items"`
}

func init() {
	ProviderLists = append(ProviderLists, &IPAMProviderList{})
	Providers = append(Providers, &IPAMProvider{})
}
