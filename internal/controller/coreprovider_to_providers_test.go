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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCoreProviderToProvidersMapper(t *testing.T) {
	g := NewWithT(t)

	testCases := []struct {
		name         string
		coreProvider client.Object
		expected     []ctrl.Request
	}{
		{
			name: "Core provider Ready condition is True",
			coreProvider: &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "core-provider",
					Namespace: testNamespaceName,
				},
				Status: operatorv1.CoreProviderStatus{
					ProviderStatus: operatorv1.ProviderStatus{
						Conditions: clusterv1.Conditions{
							{
								Type:               clusterv1.ReadyCondition,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: metav1.Now(),
								Message:            "Provider is ready",
							},
						},
					},
				},
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Namespace: testNamespaceName, Name: "preflight-checks-condition-false"}},
				{NamespacedName: types.NamespacedName{Namespace: testNamespaceName, Name: "empty-status-conditions"}},
			},
		},
		{
			name: "Core provider is not ready",
			coreProvider: &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "core-provider",
					Namespace: testNamespaceName,
				},
			},
			expected: []reconcile.Request{},
		},
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "preflight-checks-condition-false",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{},
				},
				Status: operatorv1.InfrastructureProviderStatus{
					ProviderStatus: operatorv1.ProviderStatus{
						Conditions: clusterv1.Conditions{
							{
								Type:               operatorv1.PreflightCheckCondition,
								Status:             corev1.ConditionFalse,
								LastTransitionTime: metav1.Now(),
								Reason:             operatorv1.WaitingForCoreProviderReadyReason,
								Message:            "Core provider is not ready",
							},
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "preflight-checks-condition-true",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{},
				},
				Status: operatorv1.InfrastructureProviderStatus{
					ProviderStatus: operatorv1.ProviderStatus{
						Conditions: clusterv1.Conditions{
							{
								Type:               operatorv1.PreflightCheckCondition,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: metav1.Now(),
								Message:            "Core provider is ready",
							},
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-status-conditions",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{},
				},
			},
		).
		Build()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requests := newCoreProviderToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, tc.coreProvider)
			g.Expect(requests).To(HaveLen(len(tc.expected)))
			g.Expect(requests).To(ContainElements(tc.expected))
		})
	}
}
