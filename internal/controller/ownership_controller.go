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
	"k8s.io/apimachinery/pkg/runtime/schema"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const FieldOwner = "capi-operator"

type OwnershipReconciler struct {
	// Client is a client to operate upon owned objects.
	Client client.Client

	// OwnerClinet is a dedicated client that operates on owner objects
	OwnerClient client.Client

	// LabelSelector is a common label selector between
	// owner objects and deployments handling ownership
	LabelSelector metav1.LabelSelector

	// OwnerObject is the object that holds the ownership of the owned resources
	OwnerObject client.Object

	// WaitForObjects is a list of object.ObjectList instances to propagate and wait
	// for their removal in case the owner is being deleted
	WaitForObjects []client.ObjectList

	// Finalizer is a common finalizer that will be applied to all owner and owned resources
	Finalizer string

	// PropagationPolicy describes the way how deletion process will be handled for particular objects.
	// Available modes:
	// * DeletePropagationOrphan - skip object removal on owner deletion.
	// * DeletePropagationBackground - rely solely on OwnershipReferences on owner deletion.
	// * DeletePropagationForeground - manually issue delete requests on owned resources, and wait for them to be removed.
	PropagationPolicy metav1.DeletionPropagation

	selector labels.Selector
	ownerGVK schema.GroupVersionKind
}

func (r *OwnershipReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	var err error

	r.selector, err = metav1.LabelSelectorAsSelector(&r.LabelSelector)
	if err != nil {
		return err
	}

	p, err := predicate.LabelSelectorPredicate(r.LabelSelector)
	if err != nil {
		return err
	}

	r.ownerGVK, err = apiutil.GVKForObject(r.OwnerObject, r.OwnerClient.Scheme())
	if err != nil {
		return err
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		For(r.OwnerObject).
		WithEventFilter(p).
		WithOptions(options)

	if r.ownerGVK.Kind != deploymentKind {
		builder = builder.Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestForOwner(
				r.OwnerClient.Scheme(),
				r.OwnerClient.RESTMapper(),
				r.OwnerObject))
	}

	return builder.Complete(r)
}

func (r *OwnershipReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, err error) {
	klog.Infof("Reconciling operator deployment")

	if err = r.OwnerClient.Get(ctx, req.NamespacedName, r.OwnerObject); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	defer patcher(ctx, r.OwnerClient, r.OwnerObject, &err)

	// Add finalizers first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(r.OwnerObject, r.Finalizer) {
		klog.Infof("Adding finalizer %s on the owner object %s %s",
			r.Finalizer,
			r.ownerGVK.String(),
			req.NamespacedName.String())
		controllerutil.AddFinalizer(r.OwnerObject, r.Finalizer)

		return ctrl.Result{}, nil
	}

	deployments := &appsv1.DeploymentList{}
	if err = r.OwnerClient.List(ctx, deployments, &client.ListOptions{LabelSelector: r.selector}); err != nil {
		return reconcile.Result{}, err
	}
	for _, deploy := range deployments.Items {
		deploy := deploy
		if !controllerutil.ContainsFinalizer(&deploy, r.Finalizer) {
			klog.Infof("Adding finalizer %s on the deployment %s", r.Finalizer, req.NamespacedName.String())
			controllerutil.AddFinalizer(&deploy, r.Finalizer)
			defer patcher(ctx, r.OwnerClient, &deploy, &err)

			return ctrl.Result{}, nil
		}
	}

	if r.OwnerObject.GetDeletionTimestamp().IsZero() {
		return r.reconcileNormal(ctx)
	} else {
		// Handle deletion reconciliation loop.
		return r.reconcileDelete(ctx)
	}
}

func (r *OwnershipReconciler) reconcileNormal(ctx context.Context) (_ ctrl.Result, err error) {
	// Ensure we set owner references on all managed objects for background delete propagation,
	// in cases when both clients are the same
	if r.PropagationPolicy == metav1.DeletePropagationOrphan || r.Client != r.OwnerClient {
		return
	}

	eachObj := func(obj runtime.Object) error {
		object, ok := obj.(client.Object)
		if !ok {
			return fmt.Errorf("object %s is not compatible with client.Object interface", obj.GetObjectKind())
		}

		defer patcher(ctx, r.Client, object, &err)
		object.SetOwnerReferences(util.EnsureOwnerRef(object.GetOwnerReferences(),
			metav1.OwnerReference{
				APIVersion:         r.ownerGVK.Version,
				Kind:               r.ownerGVK.Kind,
				Name:               r.OwnerObject.GetName(),
				UID:                r.OwnerObject.GetUID(),
				BlockOwnerDeletion: pointer.Bool(true),
			}))
		return nil
	}

	for _, objList := range r.WaitForObjects {
		objList := objList
		if err := r.Client.List(ctx, objList); err != nil {
			return reconcile.Result{}, err
		}

		if err := meta.EachListItem(objList, eachObj); err != nil {
			return reconcile.Result{}, fmt.Errorf("error checking for finalizer on object: %w", err)
		}
	}

	return reconcile.Result{}, nil
}

func (r *OwnershipReconciler) reconcileDelete(ctx context.Context) (_ ctrl.Result, err error) {
	res := reconcile.Result{}

	klog.Infof("Deleting owner %s replica", r.ownerGVK.String())

	deployments := &appsv1.DeploymentList{}
	if err = r.OwnerClient.List(ctx, deployments, &client.ListOptions{LabelSelector: r.selector}); err != nil {
		return res, err
	}

	// Remove finalizer from all deployments except the last one available
	for _, deploy := range deployments.Items {
		deploy := deploy
		if r.PropagationPolicy == metav1.DeletePropagationForeground && !deploy.GetDeletionTimestamp().IsZero() && deploy.Status.AvailableReplicas >= 0 {
			continue
		}

		klog.Infof(
			"There is a replica of deployment %s currently managing finalizer %s, proceeding with deletion",
			client.ObjectKeyFromObject(&deploy),
			r.Finalizer)
		defer patcher(ctx, r.Client, &deploy, &err)

		klog.Infof("Removing finalizer %s from the deployment %s",
			r.Finalizer, client.ObjectKeyFromObject(&deploy).String())
		controllerutil.RemoveFinalizer(&deploy, r.Finalizer)

		return res, nil
	}

	eachObj := func(obj runtime.Object) error {
		object, ok := obj.(client.Object)
		if !ok {
			return fmt.Errorf("object %s is not compatible with client.Object interface", obj.GetObjectKind())
		}

		gvk, err := apiutil.GVKForObject(object, r.Client.Scheme())
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
				gvk.String(),
				client.ObjectKeyFromObject(object).String(),
				r.Finalizer,
			)

			return fmt.Errorf("object %s %s still contains finalizer %s, requeuing",
				gvk.String(),
				client.ObjectKeyFromObject(object),
				r.Finalizer)
		}

		return nil
	}

	// Wait for finalizer removal from all observed objects
	if r.PropagationPolicy == metav1.DeletePropagationForeground {
		for _, objList := range r.WaitForObjects {
			objList := objList
			if err := r.Client.List(ctx, objList); err != nil {
				return res, err
			}

			if err := meta.EachListItem(objList, eachObj); err != nil {
				return res, fmt.Errorf("error checking for finalizer on object: %w", err)
			}
		}
	}

	klog.Infof("Removing finalizer %s from owner resource %s %s",
		r.Finalizer,
		r.ownerGVK.String(),
		client.ObjectKeyFromObject(r.OwnerObject).String())
	controllerutil.RemoveFinalizer(r.OwnerObject, r.Finalizer)

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
