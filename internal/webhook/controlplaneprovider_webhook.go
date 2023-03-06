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
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"sigs.k8s.io/cluster-api-operator/api/v1alpha1"
)

type ControlPlaneProviderWebhook struct {
}

func (r *ControlPlaneProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.ControlPlaneProvider{}).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-operator-cluster-x-k8s-io-v1alpha1-controlplaneprovider,mutating=false,failurePolicy=fail,groups=operator.cluster.x-k8s.io,resources=controlplaneproviders,versions=v1alpha1,name=vcontrolplaneprovider.kb.io,sideEffects=None,admissionReviewVersions=v1;v1alpha1

var _ webhook.CustomValidator = &ControlPlaneProviderWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *ControlPlaneProviderWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *ControlPlaneProviderWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *ControlPlaneProviderWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
