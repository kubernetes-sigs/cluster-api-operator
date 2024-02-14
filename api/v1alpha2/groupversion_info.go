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

// Package v1alpha2 contains API Schema definitions for the operator v1alpha2 API group
// +kubebuilder:object:generate=true
// +groupName=operator.cluster.x-k8s.io
package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "operator.cluster.x-k8s.io", Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	Providers     = []GenericProvider{}
	ProviderLists = []GenericProviderList{}
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	metav1.AddToGroupVersion(scheme, GroupVersion)

	for _, p := range Providers {
		if obj, ok := p.(runtime.Object); ok {
			scheme.AddKnownTypes(GroupVersion, obj)
		}
	}

	for _, p := range ProviderLists {
		if obj, ok := p.(runtime.Object); ok {
			scheme.AddKnownTypes(GroupVersion, obj)
		}
	}

	return nil
}
