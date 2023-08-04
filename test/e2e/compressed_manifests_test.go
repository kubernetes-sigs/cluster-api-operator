//go:build e2e
// +build e2e

/*
Copyright 2023 The Kubernetes Authors.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ociInfrastructureProviderName           = "oci"
	ociInfrastructureProviderCustomName     = "my-oci"
	ociInfrastructureProviderVersion        = "v0.12.0"
	ociInfrastructureProviderDeploymentName = "capoci-controller-manager"
	compressedAnnotation                    = "provider.cluster.x-k8s.io/compressed"
	componentsConfigMapKey                  = "components"
)

var _ = Describe("Create and delete a provider with manifests that don't fit the configmap", func() {
	var ociInfrastructureConfigMap = &corev1.ConfigMap{}

	It("should successfully create a CoreProvider", func() {
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

		Expect(k8sclient.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

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

	It("should successfully create and delete an InfrastructureProvider for OCI", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ociInfrastructureProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: ociInfrastructureProviderVersion,
				},
			},
		}

		Expect(k8sclient.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider to be ready")
		Eventually(func() bool {
			infraProvider := &operatorv1.InfrastructureProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: ociInfrastructureProviderName}
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
			key := client.ObjectKey{Namespace: operatorNamespace, Name: ociInfrastructureProviderName}
			if err := k8sclient.Get(ctx, key, infraProvider); err != nil {
				return false
			}

			if infraProvider.Status.InstalledVersion != nil && *infraProvider.Status.InstalledVersion == infraProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		By("Ensure that the created config map has correct annotation")
		cmName := fmt.Sprintf("infrastructure-%s-%s", ociInfrastructureProviderName, ociInfrastructureProviderVersion)
		key := client.ObjectKey{Namespace: operatorNamespace, Name: cmName}

		// Save config map contents to be used later.
		Expect(k8sclient.Get(ctx, key, ociInfrastructureConfigMap)).To(Succeed())

		Expect(ociInfrastructureConfigMap.GetAnnotations()[compressedAnnotation]).To(Equal("true"))

		Expect(ociInfrastructureConfigMap.BinaryData[componentsConfigMapKey]).ToNot(BeEmpty())

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: operatorNamespace,
				Name:      ociInfrastructureProviderDeploymentName,
			},
		}

		By("Waiting for the infrastructure provider deployment to be created")
		Eventually(func() bool {
			return k8sclient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment) == nil
		}, timeout).Should(Equal(true))

		Expect(k8sclient.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
		}, timeout)

		By("Waiting for the configmap to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      cmName,
				},
			},
		}, timeout)
	})

	It("should successfully create and delete an InfrastructureProvider for OCI with custom name from a pre-created ConfigMap", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ociInfrastructureProviderCustomName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"provider.cluster.x-k8s.io/name": "oci",
								"provider.cluster.x-k8s.io/type": "infrastructure",
							},
						},
					},
				},
			},
		}

		// Re-use configmap created on the previous step.
		ociInfrastructureConfigMap.ObjectMeta.UID = ""
		ociInfrastructureConfigMap.ObjectMeta.ResourceVersion = ""
		ociInfrastructureConfigMap.ObjectMeta.CreationTimestamp = metav1.Time{}
		ociInfrastructureConfigMap.ObjectMeta.OwnerReferences = nil
		Expect(k8sclient.Create(ctx, ociInfrastructureConfigMap)).To(Succeed())

		Expect(k8sclient.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider to be ready")
		Eventually(func() bool {
			infraProvider := &operatorv1.InfrastructureProvider{}
			key := client.ObjectKey{Namespace: operatorNamespace, Name: ociInfrastructureProviderCustomName}
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
			key := client.ObjectKey{Namespace: operatorNamespace, Name: ociInfrastructureProviderCustomName}
			if err := k8sclient.Get(ctx, key, infraProvider); err != nil {
				return false
			}

			if infraProvider.Status.InstalledVersion != nil && *infraProvider.Status.InstalledVersion == infraProvider.Spec.Version {
				return true
			}
			return false
		}, timeout).Should(Equal(true))

		By("Ensure that the created config map has correct annotation")
		cm := &corev1.ConfigMap{}
		cmName := fmt.Sprintf("infrastructure-%s-%s", ociInfrastructureProviderName, ociInfrastructureProviderVersion)
		key := client.ObjectKey{Namespace: operatorNamespace, Name: cmName}
		Expect(k8sclient.Get(ctx, key, cm)).To(Succeed())

		Expect(cm.GetAnnotations()[compressedAnnotation]).To(Equal("true"))

		Expect(cm.BinaryData[componentsConfigMapKey]).ToNot(BeEmpty())

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: operatorNamespace,
				Name:      ociInfrastructureProviderDeploymentName,
			},
		}

		By("Waiting for the infrastructure provider deployment to be created")
		Eventually(func() bool {
			return k8sclient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment) == nil
		}, timeout).Should(Equal(true))

		Expect(k8sclient.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: deployment,
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

		Expect(k8sclient.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      coreProviderDeploymentName,
					Namespace: operatorNamespace,
				},
			},
		}, timeout)

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(ctx, ObjectGetterInput{
			Reader: k8sclient,
			Object: coreProvider,
		}, timeout)
	})
})
