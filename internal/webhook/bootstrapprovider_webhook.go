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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

type BootstrapProviderWebhook struct{}

func (r *BootstrapProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &operatorv1.BootstrapProvider{}).
		WithValidator(r).
		WithDefaulter(r).
		Complete()
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-operator-cluster-x-k8s-io-v1alpha2-bootstrapprovider,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=operator.cluster.x-k8s.io,resources=bootstrapproviders,versions=v1alpha2,name=vbootstrapprovider.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
//+kubebuilder:webhook:verbs=create;update,path=/mutate-operator-cluster-x-k8s-io-v1alpha2-bootstrapprovider,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,matchPolicy=Equivalent,groups=operator.cluster.x-k8s.io,resources=bootstrapproviders,versions=v1alpha2,name=vbootstrapprovider.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var (
	_ admission.Validator[*operatorv1.BootstrapProvider] = &BootstrapProviderWebhook{}
	_ admission.Defaulter[*operatorv1.BootstrapProvider] = &BootstrapProviderWebhook{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *BootstrapProviderWebhook) ValidateCreate(ctx context.Context, obj *operatorv1.BootstrapProvider) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *BootstrapProviderWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj *operatorv1.BootstrapProvider) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *BootstrapProviderWebhook) ValidateDelete(_ context.Context, obj *operatorv1.BootstrapProvider) (admission.Warnings, error) {
	return nil, nil
}

// Default implements webhook.Default so a webhook will be registered for the type.
func (r *BootstrapProviderWebhook) Default(ctx context.Context, bootstrapProvider *operatorv1.BootstrapProvider) error {
	setDefaultProviderSpec(&bootstrapProvider.Spec.ProviderSpec, bootstrapProvider.Namespace)

	return nil
}
