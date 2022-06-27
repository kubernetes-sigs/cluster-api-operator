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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api-operator/controllers/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Create providers with minimal specified configuration", func() {
	It("should succefully create all providers", func() {
		k8sClient := bootstrapClusterProxy.GetClient()

		coreProvider, err := newGenericProvider(&operatorv1.CoreProvider{
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

		testProvider("CoreProvider", coreProviderDeploymentName, coreProvider, k8sClient)

		infraProvider, err := newGenericProvider(&operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      infrastructureProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    infraProviderVersion,
					SecretName: providerSecretName,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		testProvider("InfrastructureProvider", infrastructureProviderDeploymentName, infraProvider, k8sClient)

		bootstrapProvider, err := newGenericProvider(&operatorv1.BootstrapProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      boostrapProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.BootstrapProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    coreProviderVersion,
					SecretName: providerSecretName,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		testProvider("BootstrapPlane", boostrapProviderDeploymentName, bootstrapProvider, k8sClient)

		controlPlaneProvider, err := newGenericProvider(&operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      boostrapProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version:    coreProviderVersion,
					SecretName: providerSecretName,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		testProvider("ControlPlane", controlPlaneProviderDeploymentName, controlPlaneProvider, k8sClient)

		By("Deleting all providers")
		Expect(cleanupAndWait(ctx, bootstrapClusterProxy.GetClient(),
			coreProvider.GetObject(),
			infraProvider.GetObject(),
			controlPlaneProvider.GetObject(),
			bootstrapProvider.GetObject(),
		)).To(Succeed())

		By("Waiting for all deployments to be deleted")
		waitForDeploymentDeleted(operatorNamespace, coreProviderDeploymentName, k8sClient)
		waitForDeploymentDeleted(operatorNamespace, infrastructureProviderDeploymentName, k8sClient)
		waitForDeploymentDeleted(operatorNamespace, boostrapProviderDeploymentName, k8sClient)
		waitForDeploymentDeleted(operatorNamespace, controlPlaneProviderDeploymentName, k8sClient)
	})
})

func testProvider(providerType, deploymentName string, provider genericprovider.GenericProvider, k8sClient client.Client) {
	By(fmt.Sprintf("Creating %s provider", providerType))
	Expect(k8sClient.Create(ctx, provider.GetObject())).To(Succeed())

	By(fmt.Sprintf("Waiting for %s provider deployment to be ready", providerType))
	waitForDeploymentReady(operatorNamespace, deploymentName, k8sClient)

	By(fmt.Sprintf("Waiting for %s provider to be ready", providerType))
	waitForProviderCondition(provider,
		clusterv1.Condition{
			Type:   operatorv1.ProviderInstalledCondition,
			Status: corev1.ConditionTrue,
		},
		k8sClient)

	By(fmt.Sprintf("Waiting for status.IntalledVersion to be set for %s provider", providerType))
	waitForInstalledVersionSet(provider, provider.GetSpec().Version, k8sClient)
}
