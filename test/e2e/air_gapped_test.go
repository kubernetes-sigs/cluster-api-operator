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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Install Core Provider in an air-gapped environment", func() {
	It("should successfully create config maps with Core Provider manifests", func() {
		k8sclient := bootstrapClusterProxy.GetClient()

		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"core-cluster-api-v1.4.2.yaml", "core-cluster-api-v1.4.3.yaml"} {
			coreProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the core provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(coreProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Applying core provider manifests to the cluster")
		for _, cm := range configMaps {
			Expect(k8sclient.Create(ctx, &cm)).To(Succeed())
		}
	})

	It("should successfully create a CoreProvider from a config map", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"provider.cluster.x-k8s.io/name": "cluster-api",
								"provider.cluster.x-k8s.io/type": "core",
							},
						},
					},
				},
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

	It("should successfully downgrade a CoreProvider (latest -> v1.4.2)", func() {
		k8sclient := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{}
		key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
		Expect(k8sclient.Get(ctx, key, coreProvider)).To(Succeed())

		coreProvider.Spec.Version = previousCAPIVersion

		Expect(k8sclient.Update(ctx, coreProvider)).To(Succeed())

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

	It("should successfully delete config maps with Core Provider manifests", func() {
		k8sclient := bootstrapClusterProxy.GetClient()

		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"core-cluster-api-v1.4.2.yaml", "core-cluster-api-v1.4.3.yaml"} {
			coreProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the core provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(coreProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Deleting config maps with core provider manifests")
		for _, cm := range configMaps {
			Expect(k8sclient.Delete(ctx, &cm)).To(Succeed())
		}
	})
})
