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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	var err error
	deploymentPredicate, err = predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      providerLabelKey,
			Operator: metav1.LabelSelectorOpExists,
		}},
	})
	utilruntime.Must(err)
}

const providerLabelKey = "cluster.x-k8s.io/provider"

var deploymentPredicate predicate.Predicate

type ProviderHealthCheckReconciler struct {
	Client client.Client
}

type GenericProviderHealthCheckReconciler[P operatorv1.GenericProvider] struct {
	Client      client.Client
	Provider    P
	providerGVK schema.GroupVersionKind
}

func NewGenericHealthCheckReconciler[P operatorv1.GenericProvider](mgr ctrl.Manager, provider P) *GenericProviderHealthCheckReconciler[P] {
	return &GenericProviderHealthCheckReconciler[P]{
		Client:   mgr.GetClient(),
		Provider: provider,
	}
}

func (r *ProviderHealthCheckReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return kerrors.NewAggregate([]error{
		NewGenericHealthCheckReconciler(mgr, &operatorv1.CoreProvider{}).SetupWithManager(mgr, options),
		NewGenericHealthCheckReconciler(mgr, &operatorv1.InfrastructureProvider{}).SetupWithManager(mgr, options),
		NewGenericHealthCheckReconciler(mgr, &operatorv1.BootstrapProvider{}).SetupWithManager(mgr, options),
		NewGenericHealthCheckReconciler(mgr, &operatorv1.ControlPlaneProvider{}).SetupWithManager(mgr, options),
		NewGenericHealthCheckReconciler(mgr, &operatorv1.AddonProvider{}).SetupWithManager(mgr, options),
		NewGenericHealthCheckReconciler(mgr, &operatorv1.IPAMProvider{}).SetupWithManager(mgr, options),
	})
}

func (r *GenericProviderHealthCheckReconciler[P]) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	kinds, _, err := r.Client.Scheme().ObjectKinds(r.Provider)
	if err != nil {
		return err
	}

	r.providerGVK = kinds[0]

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(r.providerDeploymentPredicates())).
		WithEventFilter(deploymentPredicate).
		WithOptions(options).
		Complete(reconcile.AsReconciler(mgr.GetClient(), r))
}

func (r *GenericProviderHealthCheckReconciler[P]) Reconcile(ctx context.Context, deployment *appsv1.Deployment) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Checking provider health")

	result := ctrl.Result{}

	// There should be one owner pointing to the Provider resource.
	if err := r.Client.Get(ctx, r.getProviderKey(deployment), r.Provider); err != nil {
		// Error reading the object - requeue the request.
		return result, err
	}

	deploymentAvailableCondition := getDeploymentCondition(deployment.Status, appsv1.DeploymentAvailable)

	typedProvider := r.Provider

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
	patchHelper, err := patch.NewHelper(typedProvider, r.Client)
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

	return result, patchHelper.Patch(ctx, typedProvider, options)
}

func (r *GenericProviderHealthCheckReconciler[P]) getProviderName(deploy client.Object) string {
	for _, owner := range deploy.GetOwnerReferences() {
		if owner.Kind == r.providerGVK.Kind {
			return owner.Name
		}
	}

	return ""
}

func (r *GenericProviderHealthCheckReconciler[P]) getProviderKey(deploy client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: deploy.GetNamespace(),
		Name:      r.getProviderName(deploy),
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

func (r *GenericProviderHealthCheckReconciler[P]) providerDeploymentPredicates() predicate.Funcs {
	isProviderDeployment := func(obj runtime.Object) bool {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			panic("expected to get an of object of type appsv1.Deployment")
		}

		return r.getProviderName(deployment) != ""
	}

	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return false },
		UpdateFunc:  func(e event.UpdateEvent) bool { return isProviderDeployment(e.ObjectNew) },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
	}
}
