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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const FieldOwner = "capi-operator"

type OperatorDeploymentReconciler struct {
	client.Client
	LabelSelector     metav1.LabelSelector
	WaitForObjects    []client.ObjectList
	Finalizer         string
	PropagationPolicy metav1.DeletionPropagation

	selector labels.Selector
}

func (r *OperatorDeploymentReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	var err error

	r.selector, err = metav1.LabelSelectorAsSelector(&r.LabelSelector)
	if err != nil {
		return err
	}

	p, err := predicate.LabelSelectorPredicate(r.LabelSelector)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(p).
		WithOptions(options).
		Complete(r)
}

func (r *OperatorDeploymentReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, err error) {
	klog.Infof("Reconciling operator deployment")

	deployment := &appsv1.Deployment{}
	if err = r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	defer patcher(ctx, r.Client, deployment, &err)

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(deployment, r.Finalizer) {
		klog.Infof("Adding finalizer %s on deployment %s", r.Finalizer, req.NamespacedName.String())
		controllerutil.AddFinalizer(deployment, r.Finalizer)

		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !deployment.GetDeletionTimestamp().IsZero() {
		return r.reconcileDelete(ctx, deployment)
	}

	return reconcile.Result{}, nil
}

func (r *OperatorDeploymentReconciler) reconcileDelete(ctx context.Context, deployment *appsv1.Deployment) (_ ctrl.Result, err error) {
	res := reconcile.Result{}

	klog.Infof("Deleting deployment replica")

	deployments := &appsv1.DeploymentList{}
	if err = r.Client.List(ctx, deployments, &client.ListOptions{LabelSelector: r.selector}); err != nil {
		return res, err
	}

	// Remove finalizer from all deployments except the last one available
	for _, deploy := range deployments.Items {
		deploy := deploy
		if !deploy.GetDeletionTimestamp().IsZero() {
			continue
		}

		klog.Infof(
			"There is a replica of deployment %s currently managing finalizer %s, proceeding with deletion",
			client.ObjectKeyFromObject(&deploy),
			r.Finalizer)
		defer patcher(ctx, r.Client, &deploy, &err)

		klog.Infof("Removing finalizer %s from deployment %s",
			r.Finalizer, client.ObjectKeyFromObject(deployment).String())
		controllerutil.RemoveFinalizer(deployment, r.Finalizer)

		return res, nil
	}

	eachObj := func(obj runtime.Object) error {
		object, ok := obj.(client.Object)
		if !ok {
			return fmt.Errorf("object %s is not compatible with client.Object interface", obj.GetObjectKind())
		}

		gvk, err := apiutil.GVKForObject(object, r.Scheme())
		if err != nil {
			return err
		}

		if err := r.Client.Delete(ctx, object); client.IgnoreNotFound(err) != nil {
			klog.Errorf("Failed to issue delete on the %s object %s: %s",
				gvk,
				client.ObjectKeyFromObject(object).String(),
				err)

			return err
		}

		if controllerutil.ContainsFinalizer(object, r.Finalizer) {
			klog.Infof("Still waiting for %s object %s to remove finalizer %s",
				gvk,
				client.ObjectKeyFromObject(object).String(),
				r.Finalizer,
			)

			return fmt.Errorf("object %s %s still contains finalizer %s, requeuing",
				gvk,
				client.ObjectKeyFromObject(object),
				r.Finalizer)
		}

		return nil
	}

	// Wait for finalizer removal from all observed objects
	for _, objList := range r.WaitForObjects {
		objList := objList
		if err := r.Client.List(ctx, objList); err != nil {
			return res, err
		}

		if err := meta.EachListItem(objList, eachObj); err != nil {
			return res, fmt.Errorf("error checking for finalizer on object: %w", err)
		}
	}

	klog.Infof("Removing finalizer %s from deployment %s",
		r.Finalizer, client.ObjectKeyFromObject(deployment).String())
	controllerutil.RemoveFinalizer(deployment, r.Finalizer)

	return res, nil
}

func patcher(ctx context.Context, c client.Client, obj client.Object, reterr *error) {
	// Always attempt to update the object and status after each reconciliation.
	klog.Infof("Updating object %s", client.ObjectKeyFromObject(obj))

	patchOptions := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner(FieldOwner),
	}

	obj.SetManagedFields(nil)
	if err := c.Patch(ctx, obj, client.Apply, patchOptions...); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Info("Object %s is not found, skipping update", client.ObjectKeyFromObject(obj))
			return
		}

		*reterr = kerrors.NewAggregate([]error{*reterr, err})
		klog.Errorf("Unable to patch object: %s", *reterr)
	}
}
