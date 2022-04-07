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

var _ = Describe("Provider version conditions", func() {
	It("should set appropriate condition when provider version doesn't exist", func() {
		k8sClient := bootstrapClusterProxy.GetClient()

		provider, err := newGenericProvider(&operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    "v99.99.99",
					SecretName: providerSecretName,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		By("Creating core provider")
		Expect(k8sClient.Create(ctx, provider.GetObject()))

		By("Waiting for condition to be set")
		waitForProviderCondition(provider,
			clusterv1.Condition{
				Type:   operatorv1.PreflightCheckCondition,
				Status: corev1.ConditionFalse,
				Reason: operatorv1.CAPIVersionIncompatibilityReason,
			},
			k8sClient)

		By("Deleting core provider")
		Expect(cleanupAndWait(ctx, k8sClient, provider.GetObject())).To(Succeed())
		waitForDeploymentDeleted(operatorNamespace, coreProviderDeploymentName, k8sClient)
	})
})
