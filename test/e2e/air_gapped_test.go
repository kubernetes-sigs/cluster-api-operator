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
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	. "sigs.k8s.io/cluster-api-operator/test/framework"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Install Core Provider in an air-gapped environment", func() {
	It("should successfully create config maps with Core Provider manifests", func() {
		// Ensure that there are no Cluster API installed
		deleteClusterAPICRDs(bootstrapClusterProxy)

		bootstrapCluster := bootstrapClusterProxy.GetClient()
		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"core-cluster-api-v1.5.4.yaml", "core-cluster-api-v1.6.0.yaml"} {
			coreProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the core provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(coreProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Creating capi-system namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: capiSystemNamespace,
			},
		}
		Expect(bootstrapCluster.Create(ctx, namespace)).To(Succeed())

		By("Applying core provider manifests to the cluster")
		for _, cm := range configMaps {
			Expect(bootstrapCluster.Create(ctx, &cm)).To(Succeed())
		}
	})

	It("should successfully create a CoreProvider from a config map", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: capiSystemNamespace,
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
					Version: "v1.5.4",
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: capiSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade a CoreProvider (v1.5.4 -> latest)", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{}
		key := client.ObjectKey{Namespace: capiSystemNamespace, Name: coreProviderName}
		Expect(bootstrapCluster.Get(ctx, key, coreProvider)).To(Succeed())

		coreProvider.Spec.Version = ""

		Expect(bootstrapCluster.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: capiSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: capiSystemNamespace,
		}}

		Expect(bootstrapCluster.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderDeploymentName,
			Namespace: capiSystemNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(
			ctx, For(coreProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete config maps with Core Provider manifests", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"core-cluster-api-v1.5.4.yaml", "core-cluster-api-v1.6.0.yaml"} {
			coreProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the core provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(coreProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Deleting config maps with core provider manifests")
		for _, cm := range configMaps {
			Expect(bootstrapCluster.Delete(ctx, &cm)).To(Succeed())
		}

		By("Deleting capi-system namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: capiSystemNamespace,
			},
		}
		Expect(bootstrapCluster.Delete(ctx, namespace)).To(Succeed())
	})
})
