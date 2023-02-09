/*
Copyright 2022 The Kubernetes Authors.

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

package webhook

import (
	internalwebhook "sigs.k8s.io/cluster-api-operator/internal/webhook"
	ctrl "sigs.k8s.io/controller-runtime"
)

type BootstrapProviderWebhook struct {
}

func (r *BootstrapProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return (&internalwebhook.BootstrapProviderWebhook{}).SetupWebhookWithManager(mgr)
}

type ControlPlaneProviderWebhook struct {
}

func (r *ControlPlaneProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return (&internalwebhook.ControlPlaneProviderWebhook{}).SetupWebhookWithManager(mgr)
}

type CoreProviderWebhook struct {
}

func (r *CoreProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return (&internalwebhook.CoreProviderWebhook{}).SetupWebhookWithManager(mgr)
}

type InfrastructureProviderWebhook struct {
}

func (r *InfrastructureProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return (&internalwebhook.InfrastructureProviderWebhook{}).SetupWebhookWithManager(mgr)
}
