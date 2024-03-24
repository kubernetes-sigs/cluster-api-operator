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

package providers

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/internal/controller/phases"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericProviderReconcier[P generic.Provider] struct {
	Client          client.Client
	Config          *rest.Config
	PhaseReconciler *phases.PhaseReconciler[P, generic.Group[P]]
}

func NewGenericProviderReconcier[P generic.Provider](conn generic.Connector) generic.ProviderReconciler[P] {
	return &GenericProviderReconcier[P]{
		Client: conn.GetClient(),
		Config: conn.GetConfig(),
	}
}

func (r *GenericProviderReconcier[P]) Init() {
	r.PhaseReconciler = phases.NewPhaseReconciler[P, generic.Group[P]](r.Client)
}

// GetClient implements GenericReconciler.
func (r *GenericProviderReconcier[P]) GetClient() client.Client {
	return r.Client
}

// GetConfig implements GenericReconciler.
func (r *GenericProviderReconcier[P]) GetConfig() *rest.Config {
	return r.Config
}

// ReconcileDelete implements GenericReconciler.
func (r *GenericProviderReconcier[P]) ReconcileDelete(ctx context.Context, provider P) []generic.ReconcileFn[P, generic.Group[P]] {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Deleting provider resources")

	return generic.NewReconcileFnList(
		r.PhaseReconciler.Delete,
	)
}

// PreflightChecks implements preflight checks for GenericReconciler.
func (r *GenericProviderReconcier[P]) PreflightChecks(ctx context.Context, provider P) []generic.ReconcileFn[P, generic.Group[P]] {
	return generic.NewReconcileFnList(
		phases.PreflightChecks[P],
	)
}

// ReconcileNormal implements GenericReconciler.
func (r *GenericProviderReconcier[P]) ReconcileNormal(ctx context.Context, provider P) []generic.ReconcileFn[P, generic.Group[P]] {
	return generic.NewReconcileFnList(
		r.PhaseReconciler.InitializePhaseReconciler,
		r.PhaseReconciler.DownloadManifests,
		r.PhaseReconciler.Load,
		r.PhaseReconciler.Fetch,
		r.PhaseReconciler.Upgrade,
		r.PhaseReconciler.Install,
	)
}

// ReportStatus reports changes in status for the reconciled provider.
func (r *GenericProviderReconcier[P]) ReportStatus(ctx context.Context, provider P) []generic.ReconcileFn[P, generic.Group[P]] {
	return generic.NewReconcileFnList(
		r.PhaseReconciler.ReportStatus,
	)
}

// ClusterctlProviderType returns ProviderType for the underlying clusterctl provider
func (r *GenericProviderReconcier[P]) ClusterctlProviderType() clusterctlv1.ProviderType {
	panic("Generic Provider Reconciler has no provider type")
}

// ClusterctlProvider returns initialized underlying clusterctl provider
func (r *GenericProviderReconcier[P]) ClusterctlProvider(provider P) *clusterctlv1.Provider {
	panic("Generic Provider Reconciler has no clusterctl provider")
}

// GetProviderList returns empty typed list for provider
func (r *GenericProviderReconcier[P]) GetProviderList() generic.ProviderList {
	panic("Generic Provider Reconciler has no provider list")
}

// GenericProvider returns empty typed provider for generic reconciler
func (r *GenericProviderReconcier[P]) GenericProvider() generic.Provider {
	return reflect.New(reflect.TypeOf(*new(P)).Elem()).Interface().(P)
}

type CommonProviderReconciler[P generic.Provider] struct {
	generic.ProviderReconciler[P]
}

func NewCommonProviderReconciler[P generic.Provider](conn generic.Connector) generic.ProviderReconciler[P] {
	return &CommonProviderReconciler[P]{
		ProviderReconciler: NewGenericProviderReconcier[P](conn),
	}
}

// ReconcileNormal implements GenericReconciler.
func (r *CommonProviderReconciler[P]) PreflightChecks(
	ctx context.Context,
	provider P,
) []generic.ReconcileFn[P, generic.Group[P]] {
	return append(
		generic.NewReconcileFnList(r.waitForCoreReady),
		r.ProviderReconciler.PreflightChecks(ctx, provider)...)
}

func (r *CommonProviderReconciler[P]) waitForCoreReady(ctx context.Context, phase generic.Group[P]) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Wait for core provider to be ready before we install other providers.
	ready, err := coreProviderIsReady(ctx, phase.GetClient())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get coreProvider ready condition: %w", err)
	}

	if !ready {
		log.Info(waitingForCoreProviderReadyMessage)
		conditions.Set(phase.GetProvider(), conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.WaitingForCoreProviderReadyReason,
			clusterv1.ConditionSeverityInfo,
			waitingForCoreProviderReadyMessage,
		))

		return ctrl.Result{RequeueAfter: preflightFailedRequeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// coreProviderIsReady returns true if the core provider is ready.
func coreProviderIsReady(ctx context.Context, c client.Client) (bool, error) {
	cpl := &operatorv1.CoreProviderList{}

	if err := c.List(ctx, cpl); err != nil {
		return false, err
	}

	for _, cp := range cpl.Items {
		for _, cond := range cp.Status.Conditions {
			if cond.Type == clusterv1.ReadyCondition && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
	}

	return false, nil
}
