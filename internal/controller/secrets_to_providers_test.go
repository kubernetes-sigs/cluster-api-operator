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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProviderSecretMapper(t *testing.T) {
	g := NewWithT(t)

	otherInfraProviderNamespace := "other-namespace"
	configSecretName := "infra-provider-config"

	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithIndex(&operatorv1.InfrastructureProvider{}, configSecretNameField, configSecretNameIndexFunc).
		WithIndex(&operatorv1.InfrastructureProvider{}, configSecretNamespaceField, configSecretNamespaceIndexFunc).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-using-secret",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name: configSecretName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-using-secret-from-other-namespace",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name:      configSecretName,
							Namespace: testNamespaceName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-infra-provider-using-secret-from-other-namespace",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name:      configSecretName,
							Namespace: testNamespaceName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-with-same-secret-different-namespace",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name: configSecretName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-without-config-secret",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: nil,
					},
				},
			},
		).
		Build()

	requests := newSecretToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configSecretName,
			Namespace: testNamespaceName,
		},
	})

	g.Expect(requests).To(HaveLen(3))
	g.Expect(requests).To(ContainElements(
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespaceName, Name: "infra-provider-using-secret"}},
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: otherInfraProviderNamespace, Name: "infra-provider-using-secret-from-other-namespace"}},
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: otherInfraProviderNamespace, Name: "other-infra-provider-using-secret-from-other-namespace"}},
	))
}
