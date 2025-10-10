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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

type IPAMProviderWebhook struct{}

func (r *IPAMProviderWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&operatorv1.IPAMProvider{}).
		WithValidator(r).
		WithDefaulter(r).
		Complete()
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-operator-cluster-x-k8s-io-v1alpha3-ipamprovider,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=operator.cluster.x-k8s.io,resources=ipamproviders,versions=v1alpha3,name=vipamprovider.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
//+kubebuilder:webhook:verbs=create;update,path=/mutate-operator-cluster-x-k8s-io-v1alpha3-ipamprovider,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,matchPolicy=Equivalent,groups=operator.cluster.x-k8s.io,resources=ipamproviders,versions=v1alpha3,name=vipamprovider.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var (
	_ webhook.CustomValidator = &IPAMProviderWebhook{}
	_ webhook.CustomDefaulter = &IPAMProviderWebhook{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *IPAMProviderWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *IPAMProviderWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *IPAMProviderWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// Default implements webhook.Default so a webhook will be registered for the type.
func (r *IPAMProviderWebhook) Default(ctx context.Context, obj runtime.Object) error {
	ipamProvider, ok := obj.(*operatorv1.IPAMProvider)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a IPAMProvider but got a %T", obj))
	}

	setDefaultProviderSpec(&ipamProvider.Spec.ProviderSpec, ipamProvider.Namespace)

	return nil
}
