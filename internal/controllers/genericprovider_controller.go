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

package controllers

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GenericProviderReconciler struct {
	Provider     operatorv1.GenericProvider
	ProviderList operatorv1.GenericProviderList
	Client       client.Client
	Config       *rest.Config
}

func (r *GenericProviderReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.Provider).
		WithOptions(options).
		Complete(r)
}

func (r *GenericProviderReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Reconciling provider")

	if err := r.Client.Get(ctx, req.NamespacedName, r.Provider); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(r.Provider, r.Client)
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

		if err := patchProvider(ctx, r.Provider, patchHelper, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(r.Provider, operatorv1.ProviderFinalizer) {
		controllerutil.AddFinalizer(r.Provider, operatorv1.ProviderFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !r.Provider.GetDeletionTimestamp().IsZero() {
		return r.reconcileDelete(ctx, r.Provider)
	}

	return r.reconcile(ctx, r.Provider, r.ProviderList)
}

func patchProvider(ctx context.Context, provider operatorv1.GenericProvider, patchHelper *patch.Helper, options ...patch.Option) error {
	conds := []clusterv1.ConditionType{
		operatorv1.PreflightCheckCondition,
		operatorv1.ProviderInstalledCondition,
	}

	conditions.SetSummary(provider, conditions.WithConditions(conds...))

	options = append(options,
		patch.WithOwnedConditions{Conditions: append(conds, clusterv1.ReadyCondition)},
	)

	return patchHelper.Patch(ctx, provider, options...)
}

func (r *GenericProviderReconciler) reconcile(ctx context.Context, provider operatorv1.GenericProvider, genericProviderList operatorv1.GenericProviderList) (ctrl.Result, error) {
	reconciler := newPhaseReconciler(*r, provider, genericProviderList)
	phases := []reconcilePhaseFn{
		reconciler.preflightChecks,
		reconciler.load,
		reconciler.fetch,
		reconciler.preInstall,
		reconciler.install,
	}

	res := reconcile.Result{}
	var err error
	for _, phase := range phases {
		res, err = phase(ctx)
		if err != nil {
			se, ok := err.(*PhaseError)
			if ok {
				conditions.Set(provider, conditions.FalseCondition(se.Type, se.Reason, se.Severity, err.Error()))
			}
		}
		if !res.IsZero() || err != nil {
			// the steps are sequencial, so we must be complete before progressing.
			return res, err
		}
	}
	return res, nil
}

func (r *GenericProviderReconciler) reconcileDelete(ctx context.Context, provider operatorv1.GenericProvider) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Deleting provider resources")

	reconciler := newPhaseReconciler(*r, provider, nil)
	phases := []reconcilePhaseFn{
		reconciler.delete,
	}

	res := reconcile.Result{}
	var err error
	for _, phase := range phases {
		res, err = phase(ctx)
		if err != nil {
			se, ok := err.(*PhaseError)
			if ok {
				conditions.Set(provider, conditions.FalseCondition(se.Type, se.Reason, se.Severity, err.Error()))
			}
		}
		if !res.IsZero() || err != nil {
			// the steps are sequencial, so we must be complete before progressing.
			return res, err
		}
	}
	controllerutil.RemoveFinalizer(r.Provider, operatorv1.ProviderFinalizer)
	return res, nil
}
