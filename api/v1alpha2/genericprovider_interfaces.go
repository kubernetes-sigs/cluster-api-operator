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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenericProvider describes operations applicable to all Cluster API provider types
// (Core, Infrastructure, Bootstrap, ControlPlane, Addon, IPAM, RuntimeExtension).
// It enables the GenericProviderReconciler to manage any provider type through a
// uniform interface, embedding client.Object for Kubernetes resource semantics and
// conditions.Setter for status condition management.
//
// +kubebuilder:object:generate=false
type GenericProvider interface {
	client.Object
	conditions.Setter

	// GetSpec returns the provider's desired specification.
	GetSpec() ProviderSpec
	// SetSpec updates the provider's desired specification.
	SetSpec(in ProviderSpec)
	// GetStatus returns the provider's observed status.
	GetStatus() ProviderStatus
	// SetStatus updates the provider's observed status.
	SetStatus(in ProviderStatus)
	// GetType returns the clusterctl provider type string (e.g., "CoreProvider",
	// "InfrastructureProvider") used for provider registry lookups.
	GetType() string
	// ProviderName returns the short name of the provider as registered in the
	// clusterctl provider inventory (e.g., "cluster-api", "aws", "kubeadm").
	ProviderName() string
}

// GenericProviderList describes operations applicable to a list of GenericProvider
// objects. Each concrete provider list type (e.g., CoreProviderList) must implement
// this interface to support generic reconciliation of provider collections.
//
// +kubebuilder:object:generate=false
type GenericProviderList interface {
	// GetItems returns the list of providers as a slice of GenericProvider.
	GetItems() []GenericProvider
}
