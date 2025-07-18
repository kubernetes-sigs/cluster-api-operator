/*
Copyright 2025 The Kubernetes Authors.

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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// newConfigMapToProviderFuncMapForProviderList maps a Kubernetes ConfigMap to all the providers that reference it.
// It lists all the providers that have fetchConfig.selector that matches the ConfigMap's labels.
func newConfigMapToProviderFuncMapForProviderList(k8sClient client.Client, providerList genericprovider.GenericProviderList) handler.MapFunc {
	providerListType := fmt.Sprintf("%T", providerList)

	return func(ctx context.Context, configMap client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx).WithValues("configMap", map[string]string{"name": configMap.GetName(), "namespace": configMap.GetNamespace()}, "providerListType", providerListType)

		var requests []reconcile.Request

		// List all providers of this type
		if err := k8sClient.List(ctx, providerList, client.InNamespace(configMap.GetNamespace())); err != nil {
			log.Error(err, "failed to list providers")
			return nil
		}

		configMapLabels := labels.Set(configMap.GetLabels())

		for _, provider := range providerList.GetItems() {
			log := log.WithValues("provider", map[string]string{"name": provider.GetName(), "namespace": provider.GetNamespace()})

			// Check if provider uses fetchConfig with selector
			spec := provider.GetSpec()
			if spec.FetchConfig == nil || spec.FetchConfig.Selector == nil {
				continue
			}

			// Check if the ConfigMap matches the provider's selector
			selector, err := metav1.LabelSelectorAsSelector(spec.FetchConfig.Selector)
			if err != nil {
				log.Error(err, "failed to convert label selector")
				continue
			}

			if selector.Matches(configMapLabels) {
				log.V(1).Info("ConfigMap matches provider selector, enqueueing reconcile request")
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(provider)})
			}
		}

		return requests
	}
}
