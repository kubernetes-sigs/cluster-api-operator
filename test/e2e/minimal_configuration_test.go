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
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: coreProvider,
			Conditional: func() bool {
				for _, c := range coreProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			},
		}, timeout)

		By("Waiting for status.IntalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: coreProvider,
			Conditional: func() bool {
				return coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == coreProvider.Spec.Version
			},
		}, timeout)

		By("Checking if additional manifests are applied")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config-map",
				Namespace: operatorNamespace,
			},
		}
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: cm,
			Conditional: func() bool {
				ok, value := cm.Data["test"]
				return ok && value == "test"
			},
		}, timeout)
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
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: bootstrapProvider,
			Conditional: func() bool {
				for _, c := range bootstrapProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			},
		}, timeout)

		By("Waiting for status.IntalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: bootstrapProvider,
			Conditional: func() bool {
				return bootstrapProvider.Status.InstalledVersion != nil && *bootstrapProvider.Status.InstalledVersion == bootstrapProvider.Spec.Version
			},
		}, timeout)

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
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: cpProvider,
			Conditional: func() bool {
				for _, c := range cpProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			},
		}, timeout)

		By("Waiting for status.IntalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: cpProvider,
			Conditional: func() bool {
				return cpProvider.Status.InstalledVersion != nil && *cpProvider.Status.InstalledVersion == cpProvider.Spec.Version
			},
		}, timeout)

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
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: infraProvider,
			Conditional: func() bool {
				for _, c := range infraProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return true
					}
				}
				return false
			},
		}, timeout)

		By("Waiting for status.IntalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: infraProvider,
			Conditional: func() bool {
				return infraProvider.Status.InstalledVersion != nil && *infraProvider.Status.InstalledVersion == infraProvider.Spec.Version
			},
		}, timeout)

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

		By("Waiting for core provider to be ready and status.InstalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: coreProvider,
			Conditional: func() bool {
				for _, c := range coreProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == previousCAPIVersion
					}
				}
				return false
			},
		}, timeout)
	})

	It("should successfully upgrade a CoreProvider (v1.4.2 -> latest)", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
		}
		Expect(k8sclient.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())

		coreProvider.Spec.Version = ""

		Expect(k8sclient.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     k8sclient,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, timeout)

		By("Waiting for core provider to be ready")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: coreProvider,
			Conditional: func() bool {
				for _, c := range coreProvider.Status.Conditions {
					if c.Type == operatorv1.ProviderInstalledCondition && c.Status == corev1.ConditionTrue {
						return coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == coreProvider.Spec.Version
					}
				}
				return false
			},
		}, timeout)

		By("Waiting for the core provide status.InstalledVersion to be set")
		WaitForConditional(ctx, ObjectConditionalInput{
			Reader: k8sclient,
			Object: coreProvider,
			Conditional: func() bool {
				return coreProvider.Status.InstalledVersion != nil && *coreProvider.Status.InstalledVersion == coreProvider.Spec.Version
			},
		}, timeout)
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
