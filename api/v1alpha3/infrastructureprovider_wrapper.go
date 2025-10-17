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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ GenericProvider = &InfrastructureProvider{}

func (c *InfrastructureProvider) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *InfrastructureProvider) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func (c *InfrastructureProvider) GetSpec() ProviderSpec {
	return c.Spec.ProviderSpec
}

func (c *InfrastructureProvider) SetSpec(in ProviderSpec) {
	c.Spec.ProviderSpec = in
}

func (c *InfrastructureProvider) GetStatus() ProviderStatus {
	return c.Status.ProviderStatus
}

func (c *InfrastructureProvider) SetStatus(in ProviderStatus) {
	c.Status.ProviderStatus = in
}

func (c *InfrastructureProvider) GetType() string {
	return "infrastructure"
}

func (c *InfrastructureProvider) ProviderName() string {
	return c.GetName()
}

func (c *InfrastructureProviderList) GetItems() []GenericProvider {
	providers := make([]GenericProvider, len(c.Items))

	for index := range c.Items {
		providers[index] = &c.Items[index]
	}

	return providers
}
