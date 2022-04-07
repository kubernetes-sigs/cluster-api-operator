//go:build e2e
// +build e2e

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

package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("CoreProvider creation", func() {
	It("should set expected condition when more than one CoreProvider is created", func() {
		k8sClient := bootstrapClusterProxy.GetClient()

		coreProvider1, err := newGenericProvider(&operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    coreProviderVersion,
					SecretName: providerSecretName,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		newNamespaceName := "test-namespace"
		newNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: newNamespaceName,
			},
		}

		coreProvider2, err := newGenericProvider(&operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: newNamespaceName,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    coreProviderVersion,
					SecretName: providerSecretName,
				},
			},
		})

		Expect(err).ToNot(HaveOccurred())

		By("Creating two core providers")
		Expect(k8sClient.Create(ctx, newNamespace)).To(Succeed())
		Expect(k8sClient.Create(ctx, coreProvider1.GetObject())).To(Succeed())
		Expect(k8sClient.Create(ctx, coreProvider2.GetObject())).To(Succeed())

		By("Waiting for condition to be set")
		waitForProviderCondition(coreProvider1,
			clusterv1.Condition{
				Type:   operatorv1.PreflightCheckCondition,
				Status: corev1.ConditionFalse,
				Reason: operatorv1.MoreThanOneProviderInstanceExistsReason,
			},
			k8sClient)

		By("Deleting core providers")
		Expect(cleanupAndWait(ctx, k8sClient, coreProvider1.GetObject(), coreProvider2.GetObject())).To(Succeed())

		waitForDeploymentDeleted(operatorNamespace, coreProviderDeploymentName, k8sClient)
		waitForDeploymentDeleted(newNamespaceName, coreProviderDeploymentName, k8sClient)
	})
})
