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

type Provider interface {
	operatorv1.GenericProvider
}

type ProviderList interface {
	client.ObjectList
	operatorv1.GenericProviderList
}

type Getter interface {
	ClusterctlProviderType() clusterctlv1.ProviderType
	GenericProvider() Provider
	GetProviderList() ProviderList
}

type Connector interface {
	GetClient() client.Client
	GetConfig() *rest.Config
}

type GroupBuilder[P Provider] interface {
	Connector
	Getter
	ClusterctlProvider(provider P) *clusterctlv1.Provider
}

type Group[P Provider] interface {
	Connector
	Getter
	GetProvider() P
	GetClusterctlProvider() *clusterctlv1.Provider
}

// NewGroup is a function that creates a new group.
type NewGroup[P Provider] func(P, ProviderList, GroupBuilder[P]) Group[P]

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

func NewReconcileFnList[P Provider, G Group[P]](phaseFunc ...ReconcileFn[P, G]) []ReconcileFn[P, G] {
	return phaseFunc
}

var ProviderReconcilers = map[clusterctlv1.ProviderType]Getter{}

func GetBuilder[P Provider](_ P) GroupBuilder[P] {
	for _, reconciler := range ProviderReconcilers {
		if r, ok := reconciler.(ProviderReconciler[P]); ok {
			return r
		}
	}

	return nil
}
