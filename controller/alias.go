/*
Copyright 2025 The Kubernetes Authors.

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

/*
Package controller provides aliases for internal controller types and functions
to allow external users to interact with the core controller logic.
*/
package controller

import (
	"context"

	"k8s.io/client-go/rest"
	internalcontroller "sigs.k8s.io/cluster-api-operator/internal/controller"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	internalhealthcheck "sigs.k8s.io/cluster-api-operator/internal/controller/healthcheck"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// GenericProviderReconciler wraps the internal GenericProviderReconciler.
type GenericProviderReconciler struct {
	Provider                 genericprovider.GenericProvider
	ProviderList             genericprovider.GenericProviderList
	Client                   client.Client
	Config                   *rest.Config
	WatchConfigSecretChanges bool
}

// SetupWithManager sets up the GenericProviderReconciler with the Manager.
func (r *GenericProviderReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return (&internalcontroller.GenericProviderReconciler{
		Provider:                 r.Provider,
		ProviderList:             r.ProviderList,
		Client:                   r.Client,
		Config:                   r.Config,
		WatchConfigSecretChanges: r.WatchConfigSecretChanges,
	}).SetupWithManager(ctx, mgr, options)
}

// ProviderHealthCheckReconciler wraps the internal ProviderHealthCheckReconciler.
type ProviderHealthCheckReconciler struct {
	Client client.Client
}

// SetupWithManager sets up the health check controllers with the Manager.
func (r *ProviderHealthCheckReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return (&internalhealthcheck.ProviderHealthCheckReconciler{
		Client: r.Client,
	}).SetupWithManager(mgr, options)
}
