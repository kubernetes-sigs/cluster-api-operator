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

package v1alpha2

import (
	"sigs.k8s.io/cluster-api/util/conditions"
)

// GenericProvider interface describes operations applicable to the provider type.
//
// +kubebuilder:object:generate=false
type GenericProvider interface {
	conditions.Setter
	GetSpec() ProviderSpec
	SetSpec(in ProviderSpec)
	GetStatus() ProviderStatus
	SetStatus(in ProviderStatus)
	GetType() string
	ProviderName() string
}

// GenericProviderList interface describes operations applicable to the provider list type.
//
// +kubebuilder:object:generate=false
type GenericProviderList interface {
	GetItems() []GenericProvider
}
