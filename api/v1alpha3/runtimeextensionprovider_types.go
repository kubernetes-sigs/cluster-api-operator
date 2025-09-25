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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RuntimeExtensionProviderSpec defines the desired state of RuntimeExtensionProvider.
type RuntimeExtensionProviderSpec struct {
	ProviderSpec `json:",inline"`
}

// RuntimeExtensionProviderStatus defines the observed state of RuntimeExtensionProvider.
type RuntimeExtensionProviderStatus struct {
	ProviderStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=runtimeextensionproviders,shortName=carep,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="InstalledVersion",type="string",JSONPath=".status.installedVersion"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:storageversion

// RuntimeExtensionProvider is the Schema for the RuntimeExtensionProviders API.
type RuntimeExtensionProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuntimeExtensionProviderSpec   `json:"spec,omitempty"`
	Status RuntimeExtensionProviderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuntimeExtensionProviderList contains a list of RuntimeExtensionProviders.
type RuntimeExtensionProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuntimeExtensionProvider `json:"items"`
}

func init() {
	ProviderLists = append(ProviderLists, &RuntimeExtensionProviderList{})
	Providers = append(Providers, &RuntimeExtensionProvider{})
}
