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

package healthcheck

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ProviderHealthCheckReconciler struct {
	Client client.Client
}

const (
	providerLabelKey = "cluster.x-k8s.io/provider"
)

func (r *ProviderHealthCheckReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(providerDeploymentPredicates())).
		WithOptions(options).
		Complete(r)
}

func (r *ProviderHealthCheckReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Checking provider health")

	result := ctrl.Result{}

	deployment := &appsv1.Deployment{}

	if err := r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		// Error reading the object - requeue the request.
		return result, err
	}

	// There should be just one owner reference - to a Provider resource.
	if len(deployment.GetOwnerReferences()) != 1 {
		return result, fmt.Errorf("incorrect number of owner references for provider deployment %s", req.NamespacedName)
	}

	deploymentOwner := deployment.GetOwnerReferences()[0]

	deploymentAvailableCondition := getDeploymentCondition(deployment.Status, appsv1.DeploymentAvailable)

	typedProvider, err := r.getGenericProvider(ctx, deploymentOwner.Kind, deploymentOwner.Name, req.Namespace)
	if err != nil {
		return result, err
	}

	// Stop earlier if this provider is not fully installed yet.
	if !conditions.IsTrue(typedProvider, operatorv1.ProviderInstalledCondition) {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Compare provider's Ready condition with the deployment's Available condition and stop if they already match.
	currentReadyCondition := conditions.Get(typedProvider, clusterv1.ReadyCondition)
	if currentReadyCondition != nil && deploymentAvailableCondition != nil && currentReadyCondition.Status == deploymentAvailableCondition.Status {
		return result, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(typedProvider.GetObject(), r.Client)
	if err != nil {
		return result, err
	}

	if deploymentAvailableCondition != nil {
		conditions.Set(typedProvider, &clusterv1.Condition{
			Type:   clusterv1.ReadyCondition,
			Status: deploymentAvailableCondition.Status,
			Reason: deploymentAvailableCondition.Reason,
		})
	} else {
		conditions.Set(typedProvider, &clusterv1.Condition{
			Type:   clusterv1.ReadyCondition,
			Status: corev1.ConditionFalse,
			Reason: operatorv1.NoDeploymentAvailableConditionReason,
		})
	}

	// Don't requeue immediately if the deployment is not ready, but rather wait 5 seconds.
	if conditions.IsFalse(typedProvider, clusterv1.ReadyCondition) {
		result = ctrl.Result{RequeueAfter: 5 * time.Second}
	}

	options := patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition}}

	return result, patchHelper.Patch(ctx, typedProvider.GetObject(), options)
}

func (r *ProviderHealthCheckReconciler) getGenericProvider(ctx context.Context, providerKind, providerName, providerNamespace string) (genericprovider.GenericProvider, error) {
	switch providerKind {
	case "CoreProvider":
		provider := &operatorv1.CoreProvider{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return &genericprovider.CoreProviderWrapper{CoreProvider: provider}, nil
	case "BootstrapProvider":
		provider := &operatorv1.BootstrapProvider{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return &genericprovider.BootstrapProviderWrapper{BootstrapProvider: provider}, nil
	case "ControlPlaneProvider":
		provider := &operatorv1.ControlPlaneProvider{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return &genericprovider.ControlPlaneProviderWrapper{ControlPlaneProvider: provider}, nil
	case "InfrastructureProvider":
		provider := &operatorv1.InfrastructureProvider{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return &genericprovider.InfrastructureProviderWrapper{InfrastructureProvider: provider}, nil
	case "AddonProvider":
		provider := &operatorv1.AddonProvider{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return &genericprovider.AddonProviderWrapper{AddonProvider: provider}, nil
	default:
		return nil, fmt.Errorf("failed to cast interface for type: %s", providerKind)
	}
}

// getDeploymentCondition returns the deployment condition with the provided type.
func getDeploymentCondition(status appsv1.DeploymentStatus, condType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}

	return nil
}

func providerDeploymentPredicates() predicate.Funcs {
	isProviderDeployment := func(obj runtime.Object) bool {
		clusterOperator, ok := obj.(*appsv1.Deployment)
		if !ok {
			panic("expected to get an of object of type appsv1.Deployment")
		}

		_, found := clusterOperator.GetLabels()[providerLabelKey]

		return found
	}

	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return isProviderDeployment(e.ObjectNew) },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
	}
}
