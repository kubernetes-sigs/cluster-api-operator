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

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// newCoreProviderToProviderFuncMapForProviderList maps a ready CoreProvider object to all other provider objects.
// It lists all the providers and if its PreflightCheckCondition is not True, this object will be added to the resulting request.
// This means that notifications will only be sent to those objects that have not pass PreflightCheck.
func newCoreProviderToProviderFuncMapForProviderList(k8sClient client.Client, providerList genericprovider.GenericProviderList) handler.MapFunc {
	providerListType := fmt.Sprintf("%T", providerList)

	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		log := ctrl.LoggerFrom(ctx).WithValues("provider", map[string]string{"name": obj.GetName(), "namespace": obj.GetNamespace()}, "providerListType", providerListType)
		coreProvider, ok := obj.(*operatorv1.CoreProvider)

		if !ok {
			log.Error(fmt.Errorf("expected a %T but got a %T", operatorv1.CoreProvider{}, obj), "unable to cast object")
			return nil
		}

		// We don't want to raise events if CoreProvider is not ready yet.
		if !conditions.IsTrue(coreProvider, clusterv1.ReadyCondition) {
			return nil
		}

		var requests []reconcile.Request

		if err := k8sClient.List(ctx, providerList); err != nil {
			log.Error(err, "failed to list providers")
			return nil
		}

		for _, provider := range providerList.GetItems() {
			if !conditions.IsTrue(provider, operatorv1.PreflightCheckCondition) {
				// Raise secondary events for the providers that fail PreflightCheck.
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(provider)})
			}
		}

		return requests
	}
}
