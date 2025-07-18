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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProviderConfigMapMapper(t *testing.T) {
	g := NewWithT(t)

	namespace := "test-namespace"
	configMapLabels := map[string]string{
		"provider-components": "edge",
		"version":             "v0.1.19",
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-matching-selector",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"provider-components": "edge",
								},
							},
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-exact-match-selector",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: configMapLabels,
							},
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-non-matching-selector",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"provider-components": "aws",
								},
							},
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-without-fetchconfig",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-url-fetchconfig",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							URL: "https://github.com/example/provider/releases",
						},
					},
				},
			},
		).
		Build()

	requests := newConfigMapToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v0.1.19",
			Namespace: namespace,
			Labels:    configMapLabels,
		},
	})

	g.Expect(requests).To(HaveLen(2))
	g.Expect(requests).To(ContainElements(
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "edge-provider-with-matching-selector"}},
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "edge-provider-with-exact-match-selector"}},
	))
}

func TestProviderConfigMapMapperWithExpressions(t *testing.T) {
	g := NewWithT(t)

	namespace := "test-namespace"
	configMapLabels := map[string]string{
		"provider-components": "edge",
		"version":             "v0.1.19",
		"environment":         "production",
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-expression-selector",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"provider-components": "edge",
								},
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "environment",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{"production", "staging"},
									},
								},
							},
						},
					},
				},
			},
		).
		Build()

	requests := newConfigMapToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v0.1.19",
			Namespace: namespace,
			Labels:    configMapLabels,
		},
	})

	g.Expect(requests).To(HaveLen(1))
	g.Expect(requests).To(ContainElements(
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "edge-provider-with-expression-selector"}},
	))
}

func TestProviderConfigMapMapperNoMatches(t *testing.T) {
	g := NewWithT(t)

	namespace := "test-namespace"
	configMapLabels := map[string]string{
		"provider-components": "azure",
		"version":             "v1.9.3",
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-provider-with-non-matching-selector",
					Namespace: namespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: "v0.1.19",
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"provider-components": "edge",
								},
							},
						},
					},
				},
			},
		).
		Build()

	requests := newConfigMapToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1.9.3",
			Namespace: namespace,
			Labels:    configMapLabels,
		},
	})

	g.Expect(requests).To(BeEmpty())
}
