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
	"cmp"
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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

type ProviderHealthCheckReconciler struct{}

type GenericProviderHealthCheckReconciler struct {
	client.Client
	Provider    operatorv1.GenericProvider
	providerGVK schema.GroupVersionKind
}

func (r *ProviderHealthCheckReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return kerrors.NewAggregate([]error{
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.CoreProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.InfrastructureProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.BootstrapProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.ControlPlaneProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.AddonProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.RuntimeExtensionProvider{},
		}).SetupWithManager(mgr, options),
		(&GenericProviderHealthCheckReconciler{
			Client:   mgr.GetClient(),
			Provider: &operatorv1.IPAMProvider{},
		}).SetupWithManager(mgr, options),
	})
}

func (r *GenericProviderHealthCheckReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	kinds, _, err := r.Scheme().ObjectKinds(r.Provider)
	if err != nil {
		return err
	}

	r.providerGVK = kinds[0]

	// Provide unique name for each HC controller to avoid naming conflicts on
	// the generated name for the Deployment as a controller source.
	name := fmt.Sprintf("healthcheck-%s", r.providerGVK)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&appsv1.Deployment{}, builder.WithPredicates(predicate.NewPredicateFuncs(r.isProviderDeployment))).
		Watches(r.Provider, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, provider client.Object) []reconcile.Request {
			deploymentList := &appsv1.DeploymentList{}
			if err := r.List(ctx, deploymentList, client.InNamespace(provider.GetNamespace()), client.HasLabels{providerLabelKey}); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "Failed to list deployments for provider", "provider", client.ObjectKeyFromObject(provider))
				return nil
			}

			requests := []reconcile.Request{}

			for _, dep := range deploymentList.Items {
				if r.getProviderName(&dep) == provider.GetName() {
					requests = append(requests, reconcile.Request{
						NamespacedName: client.ObjectKeyFromObject(&dep),
					})
				}
			}

			return requests
		})).
		WithEventFilter(deploymentPredicate).
		WithOptions(options).
		Complete(reconcile.AsReconciler(mgr.GetClient(), r))
}

func (r *GenericProviderHealthCheckReconciler) Reconcile(ctx context.Context, deployment *appsv1.Deployment) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx, "provider", r.providerGVK.Kind, "providerName", r.getProviderName(deployment), "deployment", client.ObjectKeyFromObject(deployment))

	result := ctrl.Result{}

	typedProvider, ok := r.Provider.DeepCopyObject().(operatorv1.GenericProvider)
	if !ok {
		log.Error(fmt.Errorf("failed to cast provider object as GenericProvider"), "unexpected provider type")

		return result, nil
	}

	// There should be one owner pointing to the Provider resource.
	if err := r.Get(ctx, r.getProviderKey(deployment), typedProvider); err != nil {
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get provider for deployment")
		return result, err
	}

	deploymentAvailableCondition := getDeploymentCondition(deployment.Status, appsv1.DeploymentAvailable)

	// Stop earlier if this provider is not fully installed yet.
	if !conditions.IsTrue(typedProvider, operatorv1.ProviderInstalledCondition) {
		log.V(2).Info("Provider not fully installed yet, requeueing")

		return result, nil
	}

	log.Info("Checking provider health")

	// Compare provider's Ready condition with the deployment's Available condition and stop if they already match.
	currentReadyCondition := conditions.Get(typedProvider, clusterv1.ReadyCondition)
	if currentReadyCondition != nil && deploymentAvailableCondition != nil && currentReadyCondition.Status == metav1.ConditionStatus(deploymentAvailableCondition.Status) {
		log.V(5).Info("Health check conditions already in sync, skipping")

		return result, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(typedProvider, r.Client)
	if err != nil {
		return result, err
	}

	defer func() {
		if err := patchHelper.Patch(ctx, typedProvider, patch.WithOwnedConditions{Conditions: []string{clusterv1.ReadyCondition}}); err != nil {
			log.Error(err, "Failed to patch provider status")

			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	if deploymentAvailableCondition != nil {
		log.Info("Updating provider health status", "available", deploymentAvailableCondition.Status)

		conditions.Set(typedProvider, metav1.Condition{
			Type:    clusterv1.ReadyCondition,
			Status:  metav1.ConditionStatus(deploymentAvailableCondition.Status),
			Reason:  cmp.Or(deploymentAvailableCondition.Reason, operatorv1.DeploymentAvailableReason),
			Message: deploymentAvailableCondition.Message,
		})
	} else {
		conditions.Set(typedProvider, metav1.Condition{
			Type:    clusterv1.ReadyCondition,
			Status:  metav1.ConditionFalse,
			Reason:  operatorv1.NoDeploymentAvailableConditionReason,
			Message: "No minimum availability condition found on the provider deployment",
		})
	}

	if !conditions.IsTrue(typedProvider, clusterv1.ReadyCondition) {
		log.V(2).Info("Provider is not ready yet")
	}

	return result, nil
}

func (r *GenericProviderHealthCheckReconciler) getProviderName(deploy client.Object) string {
	for _, owner := range deploy.GetOwnerReferences() {
		if owner.Kind == r.providerGVK.Kind {
			return owner.Name
		}
	}

	return ""
}

func (r *GenericProviderHealthCheckReconciler) getProviderKey(deploy client.Object) types.NamespacedName {
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

func (r *GenericProviderHealthCheckReconciler) isProviderDeployment(obj client.Object) bool {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		panic("expected to get an of object of type appsv1.Deployment")
	}

	return r.getProviderName(deployment) != ""
}
