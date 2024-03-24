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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/internal/controller/phases"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ProviderControllerWrapper[P generic.Provider, R generic.ProviderReconciler[P]] struct {
	Reconciler R
	NewGroup   generic.NewGroup[P]
}

func NewProviderControllerWrapper[P generic.Provider, R generic.ProviderReconciler[P]](rec R, groupFn generic.NewGroup[P]) *ProviderControllerWrapper[P, R] {
	return &ProviderControllerWrapper[P, R]{
		Reconciler: rec,
		NewGroup:   groupFn,
	}
}

const (
	appliedSpecHashAnnotation = "operator.cluster.x-k8s.io/applied-spec-hash"
)

func (r *ProviderControllerWrapper[P, R]) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(reflect.New(reflect.TypeOf(*new(P)).Elem()).Interface().(P)).
		WithOptions(options).
		Complete(reconcile.AsReconciler(mgr.GetClient(), r))
}

func (r *ProviderControllerWrapper[P, R]) Reconcile(ctx context.Context, provider P) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Reconciling provider")

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(provider, r.Reconciler.GetClient())
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		// Always attempt to patch the object and status after each reconciliation.
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{}
		if reterr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}

		if err := patchProvider(ctx, provider, patchHelper, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer) {
		controllerutil.AddFinalizer(provider, operatorv1.ProviderFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !provider.GetDeletionTimestamp().IsZero() {
		return r.reconcileDelete(ctx, provider)
	}

	// Check if spec hash stays the same and don't go further in this case.
	specHash, err := calculateHash(provider.GetSpec())
	if err != nil {
		return ctrl.Result{}, err
	}

	if provider.GetAnnotations()[appliedSpecHashAnnotation] == specHash {
		log.Info("No changes detected, skipping further steps")

		return ctrl.Result{}, nil
	}

	res, err := r.reconcileNormal(ctx, provider)

	annotations := provider.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// Set the spec hash annotation if reconciliation was successful or reset it otherwise.
	if res.IsZero() && err == nil {
		// Recalculate spec hash in case it was changed during reconciliation process.
		specHash, err = calculateHash(provider.GetSpec())
		if err != nil {
			return ctrl.Result{}, err
		}

		annotations[appliedSpecHashAnnotation] = specHash
	} else {
		annotations[appliedSpecHashAnnotation] = ""
	}

	provider.SetAnnotations(annotations)

	return res, err
}

func patchProvider(ctx context.Context, provider operatorv1.GenericProvider, patchHelper *patch.Helper, options ...patch.Option) error {
	conds := []clusterv1.ConditionType{
		operatorv1.PreflightCheckCondition,
		operatorv1.ProviderInstalledCondition,
	}

	options = append(options, patch.WithOwnedConditions{Conditions: conds})

	return patchHelper.Patch(ctx, provider, options...)
}

func (r *ProviderControllerWrapper[P, R]) reconcileNormal(ctx context.Context, provider P) (ctrl.Result, error) {
	r.Reconciler.Init()

	phases := r.Reconciler.PreflightChecks(ctx, provider)
	phases = append(phases, r.Reconciler.ReconcileNormal(ctx, provider)...)
	phases = append(phases, r.Reconciler.ReportStatus(ctx, provider)...)

	return r.reconcilePhases(ctx, provider, phases)
}

func (r *ProviderControllerWrapper[P, R]) reconcilePhases(ctx context.Context, provider P, p []generic.ReconcileFn[P, generic.Group[P]]) (res ctrl.Result, err error) {
	providerList := r.Reconciler.GetProviderList()
	if err := r.Reconciler.GetClient().List(ctx, providerList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list providers: %w", err)
	}

	for _, phase := range p {
		res, err = phase(ctx, r.NewGroup(provider, providerList, r.Reconciler))
		if err != nil {
			var pe *phases.PhaseError
			if errors.As(err, &pe) {
				conditions.Set(provider, conditions.FalseCondition(pe.Type, pe.Reason, pe.Severity, err.Error()))
			}
		}

		if !res.IsZero() || err != nil {
			// the steps are sequential, so we must be complete before progressing.
			return res, err
		}
	}

	return res, nil
}

func (r *ProviderControllerWrapper[P, R]) reconcileDelete(ctx context.Context, provider P) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Deleting provider resources")

	r.Reconciler.Init()

	controllerutil.RemoveFinalizer(provider, operatorv1.ProviderFinalizer)

	return r.reconcilePhases(ctx, provider, r.Reconciler.ReconcileDelete(ctx, provider))
}

func calculateHash(object interface{}) (string, error) {
	jsonData, err := json.Marshal(object)
	if err != nil {
		return "", fmt.Errorf("cannot parse provider spec: %w", err)
	}

	hash := sha256.New()

	if _, err = hash.Write(jsonData); err != nil {
		return "", fmt.Errorf("cannot calculate provider spec hash: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
