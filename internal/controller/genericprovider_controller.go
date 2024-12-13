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
	"hash"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GenericProviderReconciler struct {
	Provider                 genericprovider.GenericProvider
	ProviderList             genericprovider.GenericProviderList
	Client                   client.Client
	Config                   *rest.Config
	Tracker                  *remote.ClusterCacheTracker
	WatchConfigSecretChanges bool
}

const (
	appliedSpecHashAnnotation = "operator.cluster.x-k8s.io/applied-spec-hash"
)

func (r *GenericProviderReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(r.Provider)
	if r.WatchConfigSecretChanges {
		builder.Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(newSecretToProviderFuncMapForProviderList(r.Client, r.ProviderList)),
		)
	}

	return builder.WithOptions(options).
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

	// Check if spec hash stays the same and don't go further in this case.
	specHash, err := calculateHash(ctx, r.Client, r.Provider)
	if err != nil {
		return ctrl.Result{}, err
	}

	if r.Provider.GetAnnotations()[appliedSpecHashAnnotation] == specHash {
		log.Info("No changes detected, skipping further steps")
		return ctrl.Result{}, nil
	}

	res, err := r.reconcile(ctx, r.Provider, r.ProviderList)

	annotations := r.Provider.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// Set the spec hash annotation if reconciliation was successful or reset it otherwise.
	if res.IsZero() && err == nil {
		// Recalculate spec hash in case it was changed during reconciliation process.
		specHash, err := calculateHash(ctx, r.Client, r.Provider)
		if err != nil {
			return ctrl.Result{}, err
		}

		annotations[appliedSpecHashAnnotation] = specHash
	} else {
		annotations[appliedSpecHashAnnotation] = ""
	}

	r.Provider.SetAnnotations(annotations)

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

func (r *GenericProviderReconciler) reconcile(ctx context.Context, provider genericprovider.GenericProvider, genericProviderList genericprovider.GenericProviderList) (ctrl.Result, error) {
	reconciler := newPhaseReconciler(*r, provider, genericProviderList)
	phases := []reconcilePhaseFn{
		reconciler.preflightChecks,
		reconciler.initializePhaseReconciler,
		reconciler.downloadManifests,
		reconciler.load,
		reconciler.fetch,
		reconciler.upgrade,
		reconciler.install,
		reconciler.reportStatus,
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

	controllerutil.RemoveFinalizer(provider, operatorv1.ProviderFinalizer)

	return res, nil
}

func addConfigSecretToHash(ctx context.Context, k8sClient client.Client, hash hash.Hash, provider genericprovider.GenericProvider) error {
	if provider.GetSpec().ConfigSecret != nil {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: provider.GetSpec().ConfigSecret.Namespace,
				Name:      provider.GetSpec().ConfigSecret.Name,
			},
		}
		if secret.Namespace == "" {
			secret.Namespace = provider.GetNamespace()
		}

		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		if err != nil {
			return err
		}

		err = addObjectToHash(hash, secret.Data)
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func addObjectToHash(hash hash.Hash, object interface{}) error {
	jsonData, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("cannot marshal object: %w", err)
	}

	if _, err = hash.Write(jsonData); err != nil {
		return fmt.Errorf("cannot calculate object hash: %w", err)
	}

	return nil
}

func calculateHash(ctx context.Context, k8sClient client.Client, provider genericprovider.GenericProvider) (string, error) {
	hash := sha256.New()

	err := addObjectToHash(hash, provider.GetSpec())
	if err != nil {
		return "", err
	}

	if err := addConfigSecretToHash(ctx, k8sClient, hash, provider); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
