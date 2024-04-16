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

package generic

import (
	"context"

	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Provider is an GenericProvider.
type Provider = operatorv1.GenericProvider

// ProviderList is a GenericProviderList satisfying ObjectList interface.
type ProviderList interface {
	client.ObjectList
	operatorv1.GenericProviderList
}

// Getter is a base interface for provider reconcilers.
type Getter interface {
	ClusterctlProviderType() clusterctlv1.ProviderType
	GenericProvider() Provider
	GetProviderList() ProviderList
}

// Connector is a base interface for building phase reconcile and accessing cluster via client.
type Connector interface {
	GetClient() client.Client
	GetConfig() *rest.Config
}

// GroupBuilder implementation allows to build a generic Group acting on specific provider type,
// preserving the type info.
type GroupBuilder[P Provider] interface {
	Connector
	Getter
	ClusterctlProvider(provider P) *clusterctlv1.Provider
}

// Group is a generic interface with access to typed Provider object.
// Each reconciler phase expected to work with a Provider Group.
type Group[P Provider] interface {
	Connector
	Getter
	GetProvider() P
	GetClusterctlProvider() *clusterctlv1.Provider
}

// NewGroup is a function that creates a new group.
type NewGroup[P Provider] func(P, ProviderList, GroupBuilder[P]) Group[P]

// ProviderReconciler is a reconciler methods interface related to specified provider
// The reconcile is split into 4 stages, each executed after another, and accepting any Provider object.
// Each stage will return a list of phases for controller execution, typed for defined Provider:
//
// - PreflightChecks(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
// - ReconcileNormal(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
// - ReportStatus(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
// - ReconcileDelete(ctx context.Context, provider P) []ReconcileFn[P, Group[P]].
type ProviderReconciler[P Provider] interface {
	GroupBuilder[P]
	Init()
	PreflightChecks(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
	ReconcileNormal(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
	ReportStatus(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
	ReconcileDelete(ctx context.Context, provider P) []ReconcileFn[P, Group[P]]
}

// ReconcileFn is a function that represent a phase of the reconciliation.
type ReconcileFn[P Provider, G Group[P]] func(context.Context, G) (reconcile.Result, error)

// NewReconcileFnList created a list of reconcile phases, with a typed group working with a defined provider only
// Example:
//
//	generic.NewReconcileFnList(r.corePreflightChecks) // Will only compile when passed to core provider reconciler working on CoreProvider
//
//	func (r *CoreProviderReconciler) corePreflightChecks(ctx context.Context, phase generic.Group[*operatorv1.CoreProvider]) (ctrl.Result, error) {
//			var p *operatorv1.CoreProvider
//			// getting actual core provider instead of interface for resource specific operations or validation
//			p = phase.GetProvider() // this works
//	}
func NewReconcileFnList[P Provider, G Group[P]](phaseFunc ...ReconcileFn[P, G]) []ReconcileFn[P, G] {
	return phaseFunc
}

// ProviderReconcilers is a storage of registered provider reconcilers on controller startup.
// It is used to access reconciler specific methods, allowing to map Clusterctl provider type
// on an actual Provider object, which represents it.
var ProviderReconcilers = map[clusterctlv1.ProviderType]Getter{}

// GetBuilder provides an initialized reconciler to fetch component in the domail of provider, like
// provider list type, clusterctl provider, etc. without need to maintain an evergrowing switch statement.
func GetBuilder[P Provider](_ P) GroupBuilder[P] {
	for _, reconciler := range ProviderReconcilers {
		if r, ok := reconciler.(ProviderReconciler[P]); ok {
			return r
		}
	}

	return nil
}
