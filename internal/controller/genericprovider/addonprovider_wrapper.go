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

package genericprovider

import (
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AddonProviderWrapper struct {
	*operatorv1.AddonProvider
}

func (b *AddonProviderWrapper) GetConditions() clusterv1.Conditions {
	return b.Status.Conditions
}

func (b *AddonProviderWrapper) SetConditions(conditions clusterv1.Conditions) {
	b.Status.Conditions = conditions
}

func (b *AddonProviderWrapper) GetSpec() operatorv1.ProviderSpec {
	return b.Spec.ProviderSpec
}

func (b *AddonProviderWrapper) SetSpec(in operatorv1.ProviderSpec) {
	b.Spec.ProviderSpec = in
}

func (b *AddonProviderWrapper) GetStatus() operatorv1.ProviderStatus {
	return b.Status.ProviderStatus
}

func (b *AddonProviderWrapper) SetStatus(in operatorv1.ProviderStatus) {
	b.Status.ProviderStatus = in
}

func (b *AddonProviderWrapper) GetObject() client.Object {
	return b.AddonProvider
}

func (b *AddonProviderWrapper) GetType() string {
	return "addon"
}

type AddonProviderListWrapper struct {
	*operatorv1.AddonProviderList
}

func (b *AddonProviderListWrapper) GetItems() []GenericProvider {
	providers := []GenericProvider{}

	for index := range b.Items {
		providers = append(providers, &AddonProviderWrapper{&b.Items[index]})
	}

	return providers
}

func (b *AddonProviderListWrapper) GetObject() client.ObjectList {
	return b.AddonProviderList
}
