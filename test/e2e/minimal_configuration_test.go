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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Create, upgrade, downgrade and delete providers with minimal specified configuration", func() {
	It("should successfully create a CoreProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()

		additionalManifestsCMName := "additional-manifests"
		additionalManifests := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      additionalManifestsCMName,
				Namespace: operatorNamespace,
			},
			Data: map[string]string{
				"manifests": `
---				
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config-map
  namespace: capi-operator-system
data: 
  test: test
`,
			},
		}

		Expect(k8sclient.Create(ctx, additionalManifests)).To(Succeed())

		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					AdditionalManifestsRef: &operatorv1.ConfigmapReference{
						Name: additionalManifests.Name,
					},
				},
			},
		}

		Expect(k8sclient.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, timeout)

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

			if coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == coreProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		By("Checking if additional manifests are applied")
		Eventually(func() bool {
			cm := &corev1.ConfigMap{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: "test-config-map"}
			if err := k8sclient.Get(ctx, key, cm); err != nil {
				return false
			}
			if cm.Data["test"] == "test" {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should successfully create and delete a BootstrapProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.BootstrapProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{},
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapProviderDeploymentName,
				Namespace: operatorNamespace,
			},
		}

		Expect(k8sclient.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: deployment,
		}, timeout)

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

			if bootstrapProvider.Status.InstalledVersion != nil && *bootstrapProvider.Status.InstalledVersion == bootstrapProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		Expect(k8sclient.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
		})
	})

	It("should successfully create and delete a ControlPlaneProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		cpProvider := &operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cpProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{},
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cpProviderDeploymentName,
				Namespace: operatorNamespace,
			},
		}

		Expect(k8sclient.Create(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: deployment,
		}, timeout)

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

			if cpProvider.Status.InstalledVersion != nil && *cpProvider.Status.InstalledVersion == cpProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		Expect(k8sclient.Delete(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
		})
	})

	It("should successfully create and delete an InfrastructureProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      infraProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{},
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      infraProviderDeploymentName,
				Namespace: operatorNamespace,
			},
		}
		Expect(k8sclient.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: deployment,
		}, timeout)

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

			if infraProvider.Status.InstalledVersion != nil && *infraProvider.Status.InstalledVersion == infraProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		Expect(k8sclient.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
		})
	})

	It("should successfully downgrade a CoreProvider (latest -> v1.4.2)", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{}
		key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
		Expect(k8sclient.Get(ctx, key, coreProvider)).To(Succeed())

		coreProvider.Spec.Version = previousCAPIVersion

		Expect(k8sclient.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, timeout)

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

			if coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == previousCAPIVersion {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should successfully upgrade a CoreProvider (v1.4.2 -> latest)", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{}
		key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
		Expect(k8sclient.Get(ctx, key, coreProvider)).To(Succeed())

		coreProvider.Spec.Version = ""

		Expect(k8sclient.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, timeout)

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

			if coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == coreProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))
	})

	It("should successfully delete a CoreProvider", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{},
			},
		}
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: operatorNamespace,
				Name:      coreProviderDeploymentName,
			},
		}

		Expect(k8sclient.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
		})

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: coreProvider,
		})
	})
})
