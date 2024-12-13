/*
Copyright 2024 The Kubernetes Authors.

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

	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// newSecretToProviderFuncMapForProviderList maps a Kubernetes secret to all the providers that reference it.
// It lists all the providers and checks whether the secret is referenced as the provider's config secret.
// If the providers references a secret without a namespace, it will assume the secret is in the same namespace as the provider.
func newSecretToProviderFuncMapForProviderList(k8sClient client.Client, providerList genericprovider.GenericProviderList) handler.MapFunc {
	providerListType := fmt.Sprintf("%t", providerList)

	return func(ctx context.Context, secret client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx).WithValues("secret", map[string]string{"name": secret.GetName(), "namespace": secret.GetNamespace()}, "providerListType", providerListType)

		var requests []reconcile.Request

		if err := k8sClient.List(ctx, providerList); err != nil {
			log.Error(err, "failed to list providers")
			return nil
		}

		for _, provider := range providerList.GetItems() {
			log = log.WithValues("provider", map[string]string{"name": provider.GetName(), "namespace": provider.GetNamespace()})

			spec := provider.GetSpec()

			if spec.ConfigSecret == nil {
				log.V(5).Info("Provider does not reference a config secret")
				continue
			}

			configNamespace := spec.ConfigSecret.Namespace
			if configNamespace == "" {
				log.V(5).Info("Provider configSecret namespace is empty, using provider namespace")

				configNamespace = provider.GetNamespace()
			}

			if configNamespace == secret.GetNamespace() && spec.ConfigSecret.Name == secret.GetName() {
				log.V(2).Info("Found provider using config secret")

				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(provider)})
			}
		}

		return requests
	}
}
