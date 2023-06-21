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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
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
	Provider     client.Object
	ProviderList client.ObjectList
	Client       client.Client
	Config       *rest.Config
}

const (
	appliedSpecHashAnnotation = "operator.cluster.x-k8s.io/applied-spec-hash"
)

func (r *GenericProviderReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.Provider).
		WithOptions(options).
		Complete(r)
}

func (r *GenericProviderReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Reconciling provider")

	typedProvider, err := r.newGenericProvider()
	if err != nil {
		return ctrl.Result{}, err
	}

	typedProviderList, err := r.newGenericProviderList()
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Get(ctx, req.NamespacedName, typedProvider.GetObject()); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(typedProvider.GetObject(), r.Client)
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

		if err := patchProvider(ctx, typedProvider, patchHelper, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(typedProvider.GetObject(), operatorv1.ProviderFinalizer) {
		controllerutil.AddFinalizer(typedProvider.GetObject(), operatorv1.ProviderFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !typedProvider.GetDeletionTimestamp().IsZero() {
		return r.reconcileDelete(ctx, typedProvider)
	}

	// Check if spec hash stays the same and don't go further in this case.
	specHash, err := calculateHash(typedProvider.GetSpec())
	if err != nil {
		return ctrl.Result{}, err
	}

	if typedProvider.GetAnnotations()[appliedSpecHashAnnotation] == specHash {
		log.Info("No changes detected, skipping further steps")

		return ctrl.Result{}, nil
	}

	res, err := r.reconcile(ctx, typedProvider, typedProviderList)

	annotations := typedProvider.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// Set the spec hash annotation if reconciliation was successful or reset it otherwise.
	if res.IsZero() && err == nil {
		annotations[appliedSpecHashAnnotation] = specHash
	} else {
		annotations[appliedSpecHashAnnotation] = ""
	}

	typedProvider.SetAnnotations(annotations)

	return res, err
}

func patchProvider(ctx context.Context, provider genericprovider.GenericProvider, patchHelper *patch.Helper, options ...patch.Option) error {
	conds := []clusterv1.ConditionType{
		operatorv1.PreflightCheckCondition,
		operatorv1.ProviderInstalledCondition,
	}

	conditions.SetSummary(provider, conditions.WithConditions(conds...))

	options = append(options,
		patch.WithOwnedConditions{Conditions: append(conds, clusterv1.ReadyCondition)},
	)

	return patchHelper.Patch(ctx, provider.GetObject(), options...)
}

func (r *GenericProviderReconciler) reconcile(ctx context.Context, provider genericprovider.GenericProvider, genericProviderList genericprovider.GenericProviderList) (ctrl.Result, error) {
	reconciler := newPhaseReconciler(*r, provider, genericProviderList)
	phases := []reconcilePhaseFn{
		reconciler.preflightChecks,
		reconciler.initializePhaseReconciler,
		reconciler.downloadManifests,
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
			var pe *PhaseError
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

func (r *GenericProviderReconciler) reconcileDelete(ctx context.Context, provider genericprovider.GenericProvider) (ctrl.Result, error) {
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
			var pe *PhaseError
			if errors.As(err, &pe) {
				conditions.Set(provider, conditions.FalseCondition(pe.Type, pe.Reason, pe.Severity, err.Error()))
			}
		}

		if !res.IsZero() || err != nil {
			// the steps are sequential, so we must be complete before progressing.
			return res, err
		}
	}

	controllerutil.RemoveFinalizer(provider.GetObject(), operatorv1.ProviderFinalizer)

	return res, nil
}

func (r *GenericProviderReconciler) newGenericProvider() (genericprovider.GenericProvider, error) {
	switch r.Provider.(type) {
	case *operatorv1.CoreProvider:
		return &genericprovider.CoreProviderWrapper{CoreProvider: &operatorv1.CoreProvider{}}, nil
	case *operatorv1.BootstrapProvider:
		return &genericprovider.BootstrapProviderWrapper{BootstrapProvider: &operatorv1.BootstrapProvider{}}, nil
	case *operatorv1.ControlPlaneProvider:
		return &genericprovider.ControlPlaneProviderWrapper{ControlPlaneProvider: &operatorv1.ControlPlaneProvider{}}, nil
	case *operatorv1.InfrastructureProvider:
		return &genericprovider.InfrastructureProviderWrapper{InfrastructureProvider: &operatorv1.InfrastructureProvider{}}, nil
	default:
		providerKind := reflect.Indirect(reflect.ValueOf(r.Provider)).Type().Name()
		failedToCastInterfaceErr := fmt.Errorf("failed to cast interface for type: %s", providerKind)

		return nil, failedToCastInterfaceErr
	}
}

func (r *GenericProviderReconciler) newGenericProviderList() (genericprovider.GenericProviderList, error) {
	switch r.ProviderList.(type) {
	case *operatorv1.CoreProviderList:
		return &genericprovider.CoreProviderListWrapper{CoreProviderList: &operatorv1.CoreProviderList{}}, nil
	case *operatorv1.BootstrapProviderList:
		return &genericprovider.BootstrapProviderListWrapper{BootstrapProviderList: &operatorv1.BootstrapProviderList{}}, nil
	case *operatorv1.ControlPlaneProviderList:
		return &genericprovider.ControlPlaneProviderListWrapper{ControlPlaneProviderList: &operatorv1.ControlPlaneProviderList{}}, nil
	case *operatorv1.InfrastructureProviderList:
		return &genericprovider.InfrastructureProviderListWrapper{InfrastructureProviderList: &operatorv1.InfrastructureProviderList{}}, nil
	default:
		providerKind := reflect.Indirect(reflect.ValueOf(r.ProviderList)).Type().Name()
		failedToCastInterfaceErr := fmt.Errorf("failed to cast interface for type: %s", providerKind)

		return nil, failedToCastInterfaceErr
	}
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
