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

package v1alpha2

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ GenericProvider = &ControlPlaneProvider{}

func (c *ControlPlaneProvider) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *ControlPlaneProvider) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func (c *ControlPlaneProvider) GetSpec() ProviderSpec {
	return c.Spec.ProviderSpec
}

func (c *ControlPlaneProvider) SetSpec(in ProviderSpec) {
	c.Spec.ProviderSpec = in
}

func (c *ControlPlaneProvider) GetStatus() ProviderStatus {
	return c.Status.ProviderStatus
}

func (c *ControlPlaneProvider) SetStatus(in ProviderStatus) {
	c.Status.ProviderStatus = in
}

func (c *ControlPlaneProvider) GetType() string {
	return "controlplane"
}

func (c *ControlPlaneProvider) ProviderName() string {
	return c.GetName()
}

func (c *ControlPlaneProviderList) GetItems() []GenericProvider {
	providers := make([]GenericProvider, len(c.Items))

	for index := range c.Items {
		providers[index] = &c.Items[index]
	}

	return providers
}
