package controller

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

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SecretReconciler struct {
	ProviderLists []genericprovider.GenericProviderList
	Client        client.Client
}

const (
	observedSpecHashAnnotation = "operator.cluster.x-k8s.io/observed-spec-hash"
)

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		WithOptions(options).
		Complete(r)
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Reconciling provider")

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      req.Name,
		},
	}

	err := r.Client.Get(ctx, req.NamespacedName, secret)
	if ctrlclient.IgnoreNotFound(err) != nil {
		return reconcile.Result{}, err
	}

	hash := ""
	if !apierrors.IsNotFound(err) {
		hash, err = calculateHash(secret.Data)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, group := range r.ProviderLists {
		g, ok := group.(client.ObjectList)
		if !ok {
			continue
		}

		if err := r.Client.List(ctx, g); err != nil {
			return reconcile.Result{}, err
		}

		for _, p := range group.GetItems() {
			if p.GetSpec().ConfigSecret != nil {
				configNamespace := p.GetSpec().ConfigSecret.Namespace
				if configNamespace == "" {
					configNamespace = p.GetNamespace()
				}
				if configNamespace == req.Namespace && p.GetSpec().ConfigSecret.Name == req.Name {
					patched, ok := p.DeepCopyObject().(client.Object)
					if !ok {
						// todo: just log
					} else {
						annotations := patched.GetAnnotations()
						if annotations == nil {
							annotations = map[string]string{}
						}
						if annotations[observedSpecHashAnnotation] != hash {
							annotations[observedSpecHashAnnotation] = hash
							patched.SetAnnotations(annotations)
							err := r.Client.Patch(ctx, patched, client.MergeFrom(p))
							if err != nil {
								return reconcile.Result{}, err
							}
						}
					}
				}
			}
		}
	}
	return reconcile.Result{}, nil
}
