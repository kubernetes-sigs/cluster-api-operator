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

	"github.com/Masterminds/goutils"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	configSecretNameField      = "spec.configSecret.name"      //nolint:gosec
	configSecretNamespaceField = "spec.configSecret.namespace" //nolint:gosec
)

// newSecretToProviderFuncMapForProviderList maps a Kubernetes secret to all the providers that reference it.
// It lists all the providers matching spec.configSecret.name values with the secret name querying by index.
// If the provider references a secret without a namespace, it will assume the secret is in the same namespace as the provider.
func newSecretToProviderFuncMapForProviderList(k8sClient client.Client, providerList genericprovider.GenericProviderList) handler.MapFunc {
	providerListType := fmt.Sprintf("%T", providerList)

	return func(ctx context.Context, secret client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx).WithValues("secret", map[string]string{"name": secret.GetName(), "namespace": secret.GetNamespace()}, "providerListType", providerListType)

		var requests []reconcile.Request

		configSecretMatcher := client.MatchingFields{configSecretNameField: secret.GetName(), configSecretNamespaceField: secret.GetNamespace()}
		if err := k8sClient.List(ctx, providerList, configSecretMatcher); err != nil {
			log.Error(err, "failed to list providers")
			return nil
		}

		for _, provider := range providerList.GetItems() {
			log = log.WithValues("provider", map[string]string{"name": provider.GetName(), "namespace": provider.GetNamespace()})
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(provider)})
		}

		return requests
	}
}

// configSecretNameIndexFunc is indexing config Secret name field.
var configSecretNameIndexFunc = func(obj client.Object) []string {
	provider, ok := obj.(operatorv1.GenericProvider)
	if !ok || provider.GetSpec().ConfigSecret == nil {
		return nil
	}

	return []string{provider.GetSpec().ConfigSecret.Name}
}

// configSecretNamespaceIndexFunc is indexing config Secret namespace field.
var configSecretNamespaceIndexFunc = func(obj client.Object) []string {
	provider, ok := obj.(operatorv1.GenericProvider)
	if !ok || provider.GetSpec().ConfigSecret == nil {
		return nil
	}

	return []string{goutils.DefaultString(provider.GetSpec().ConfigSecret.Namespace, provider.GetNamespace())}
}
