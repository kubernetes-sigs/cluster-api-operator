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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Create providers with minimal specified configuration", func() {
	It("should succefully create a CoreProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: capiVersion,
				},
			},
		}

		Expect(k8sclient.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		Eventually(func() bool {
			isReady, err := waitForDeployment(k8sclient, ctx, coreProviderDeploymentName, operatorNamespace)
			if err != nil {
				return false
			}
			return isReady
		}, timeout).Should(Equal(true))

		By("Waiting for core provider to be ready")
		Eventually(func() bool {
			coreProvider := &operatorv1.CoreProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
			if err := k8sclient.Get(ctx, key, coreProvider); err != nil {
				return false
			}

			for _, c := range coreProvider.Status.Conditions {
				if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout).Should(Equal(true))

		By("Waiting for status.IntalledVersion to be set")
		Eventually(func() bool {
			coreProvider := &operatorv1.CoreProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
			if err := k8sclient.Get(ctx, key, coreProvider); err != nil {
				return false
			}

			if coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == capiVersion {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should succefully create a BootstrapProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.BootstrapProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: capiVersion,
				},
			},
		}

		Expect(k8sclient.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		Eventually(func() bool {
			isReady, err := waitForDeployment(k8sclient, ctx, bootstrapProviderDeploymentName, operatorNamespace)
			if err != nil {
				return false
			}
			return isReady
		}, timeout).Should(Equal(true))

		By("Waiting for bootstrap provider to be ready")
		Eventually(func() bool {
			bootstrapProvider := &operatorv1.BootstrapProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: bootstrapProviderName}
			if err := k8sclient.Get(ctx, key, bootstrapProvider); err != nil {
				return false
			}

			for _, c := range bootstrapProvider.Status.Conditions {
				if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout).Should(Equal(true))

		By("Waiting for status.IntalledVersion to be set")
		Eventually(func() bool {
			bootstrapProvider := &operatorv1.BootstrapProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: bootstrapProviderName}
			if err := k8sclient.Get(ctx, key, bootstrapProvider); err != nil {
				return false
			}

			if bootstrapProvider.Status.InstalledVersion != nil && *bootstrapProvider.Status.InstalledVersion == capiVersion {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should succefully create a ControlPlaneProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		cpProvider := &operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cpProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: capiVersion,
				},
			},
		}

		Expect(k8sclient.Create(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be ready")
		Eventually(func() bool {
			isReady, err := waitForDeployment(k8sclient, ctx, cpProviderDeploymentName, operatorNamespace)
			if err != nil {
				return false
			}
			return isReady
		}, timeout).Should(Equal(true))

		By("Waiting for the control plane provider to be ready")
		Eventually(func() bool {
			cpProvider := &operatorv1.ControlPlaneProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: cpProviderName}
			if err := k8sclient.Get(ctx, key, cpProvider); err != nil {
				return false
			}

			for _, c := range cpProvider.Status.Conditions {
				if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout).Should(Equal(true))

		By("Waiting for status.IntalledVersion to be set")
		Eventually(func() bool {
			cpProvider := &operatorv1.ControlPlaneProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: cpProviderName}
			if err := k8sclient.Get(ctx, key, cpProvider); err != nil {
				return false
			}

			if cpProvider.Status.InstalledVersion != nil && *cpProvider.Status.InstalledVersion == capiVersion {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should succefully create a InfrastructureProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      infraProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: capiVersion,
				},
			},
		}

		Expect(k8sclient.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be ready")
		Eventually(func() bool {
			isReady, err := waitForDeployment(k8sclient, ctx, infraProviderDeploymentName, operatorNamespace)
			if err != nil {
				return false
			}
			return isReady
		}, timeout).Should(Equal(true))

		By("Waiting for the infrastructure provider to be ready")
		Eventually(func() bool {
			infraProvider := &operatorv1.InfrastructureProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: infraProviderName}
			if err := k8sclient.Get(ctx, key, infraProvider); err != nil {
				return false
			}

			for _, c := range infraProvider.Status.Conditions {
				if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout).Should(Equal(true))

		By("Waiting for status.IntalledVersion to be set")
		Eventually(func() bool {
			infraProvider := &operatorv1.InfrastructureProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: infraProviderName}
			if err := k8sclient.Get(ctx, key, infraProvider); err != nil {
				return false
			}

			if infraProvider.Status.InstalledVersion != nil && *infraProvider.Status.InstalledVersion == capiVersion {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})
})
