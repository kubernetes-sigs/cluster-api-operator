//go:build e2e

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

var _ = Describe("Install Controlplane, Core, Bootstrap Providers in an air-gapped environment", func() {
	It("should successfully create config maps with Core and Bootstrap Provider manifests", func() {
		// Ensure that there are no Cluster API installed
		deleteClusterAPICRDs(bootstrapClusterProxy)

		bootstrapCluster := bootstrapClusterProxy.GetClient()
		configMaps := []corev1.ConfigMap{}
		configMapFiles := []string{
			"core-cluster-api-v1.7.7.yaml",
			"core-cluster-api-v1.8.0.yaml",
			"bootstrap-kubeadm-v1.7.7.yaml",
			"bootstrap-kubeadm-v1.8.0.yaml",
			"controlplane-kubeadm-v1.7.7.yaml",
			"controlplane-kubeadm-v1.8.0.yaml",
		}

		for _, fileName := range configMapFiles {
			providerComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(providerComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Creating provider namespaces")
		for _, namespaceName := range []string{capkbSystemNamespace, capkcpSystemNamespace, capiSystemNamespace} {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(bootstrapCluster.Create(ctx, namespace)).To(Succeed())
		}

		By("Applying provider manifests to the cluster")
		for _, cm := range configMaps {
			Expect(bootstrapCluster.Create(ctx, &cm)).To(Succeed())
		}
	})

	It("should successfully create a BootstrapProvider from a config map", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customProviderName,
				Namespace: capkbSystemNamespace,
			},
			Spec: operatorv1.BootstrapProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"provider.cluster.x-k8s.io/name":    "kubeadm",
								"provider.cluster.x-k8s.io/type":    "bootstrap",
								"provider.cluster.x-k8s.io/version": "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: bootstrapProviderDeploymentName, Namespace: capkbSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&bootstrapProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(bootstrapProvider.Status.InstalledVersion, ptr.To(bootstrapProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
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
								"provider.cluster.x-k8s.io/name":    "cluster-api",
								"provider.cluster.x-k8s.io/type":    "core",
								"provider.cluster.x-k8s.io/version": "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
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

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create a ControlPlaneProvider from a config map", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		controlPlaneProvider := &operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customProviderName,
				Namespace: capkcpSystemNamespace,
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"provider.cluster.x-k8s.io/name":    "kubeadm",
								"provider.cluster.x-k8s.io/type":    "controlplane",
								"provider.cluster.x-k8s.io/version": "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, controlPlaneProvider)).To(Succeed())

		By("Waiting for the controlplane provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: cpProviderDeploymentName, Namespace: capkcpSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for controlplane provider to be ready")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&controlPlaneProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(controlPlaneProvider.Status.InstalledVersion, ptr.To(controlPlaneProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade BootstrapProvider, ControlPlaneProvider, CoreProvider (v1.7.7 -> v1.8.0)", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()

		bootstrapProvider := &operatorv1.BootstrapProvider{}
		coreProvider := &operatorv1.CoreProvider{}
		controlPlaneProvider := &operatorv1.ControlPlaneProvider{}

		bootstrapKey := client.ObjectKey{Namespace: capkbSystemNamespace, Name: customProviderName}
		Expect(bootstrapCluster.Get(ctx, bootstrapKey, bootstrapProvider)).To(Succeed())

		coreKey := client.ObjectKey{Namespace: capiSystemNamespace, Name: coreProviderName}
		Expect(bootstrapCluster.Get(ctx, coreKey, coreProvider)).To(Succeed())

		cpKey := client.ObjectKey{Namespace: capkcpSystemNamespace, Name: customProviderName}
		Expect(bootstrapCluster.Get(ctx, cpKey, controlPlaneProvider)).To(Succeed())

		bootstrapProvider.Spec.Version = nextCAPIVersion
		bootstrapProvider.Spec.FetchConfig.Selector.MatchLabels["provider.cluster.x-k8s.io/version"] = nextCAPIVersion
		coreProvider.Spec.Version = nextCAPIVersion
		coreProvider.Spec.FetchConfig.Selector.MatchLabels["provider.cluster.x-k8s.io/version"] = nextCAPIVersion
		controlPlaneProvider.Spec.Version = nextCAPIVersion
		controlPlaneProvider.Spec.FetchConfig.Selector.MatchLabels["provider.cluster.x-k8s.io/version"] = nextCAPIVersion

		Expect(bootstrapCluster.Update(ctx, bootstrapProvider)).To(Succeed())
		Expect(bootstrapCluster.Update(ctx, coreProvider)).To(Succeed())
		Expect(bootstrapCluster.Update(ctx, controlPlaneProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: bootstrapProviderDeploymentName, Namespace: capkbSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&bootstrapProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider status.InstalledVersion to be set")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(bootstrapProvider.Status.InstalledVersion, ptr.To(bootstrapProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: capiSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the controlplane provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: cpProviderDeploymentName, Namespace: capkcpSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for controlplane provider to be ready")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(controlPlaneProvider.Status.InstalledVersion, ptr.To(controlPlaneProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete a BootstrapProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      customProviderName,
			Namespace: capkbSystemNamespace,
		}}

		Expect(bootstrapCluster.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderDeploymentName,
			Namespace: capkbSystemNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the bootstrap provider object to be deleted")
		WaitForDelete(
			ctx, For(bootstrapProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete config maps with Bootstrap Provider manifests", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"bootstrap-kubeadm-v1.7.7.yaml", "bootstrap-kubeadm-v1.8.0.yaml"} {
			bootstrapProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the bootstrap provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(bootstrapProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Deleting config maps with bootstrap provider manifests")
		for _, cm := range configMaps {
			Expect(bootstrapCluster.Delete(ctx, &cm)).To(Succeed())
		}

		By("Deleting capkb-system namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: capkbSystemNamespace,
			},
		}
		Expect(bootstrapCluster.Delete(ctx, namespace)).To(Succeed())
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

		for _, fileName := range []string{"core-cluster-api-v1.7.7.yaml", "core-cluster-api-v1.8.0.yaml"} {
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

	It("should successfully delete a ControlPlaneProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		ControlPlaneProvider := &operatorv1.ControlPlaneProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      customProviderName,
			Namespace: capkcpSystemNamespace,
		}}

		Expect(bootstrapCluster.Delete(ctx, ControlPlaneProvider)).To(Succeed())

		By("Waiting for the controlplane provider deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderDeploymentName,
			Namespace: capkcpSystemNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the controlplane provider object to be deleted")
		WaitForDelete(
			ctx, For(ControlPlaneProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete config maps with ControlPlane Provider manifests", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		configMaps := []corev1.ConfigMap{}

		for _, fileName := range []string{"controlplane-kubeadm-v1.7.7.yaml", "controlplane-kubeadm-v1.8.0.yaml"} {
			controlPlaneProviderComponents, err := os.ReadFile(customManifestsFolder + fileName)
			Expect(err).ToNot(HaveOccurred(), "Failed to read the controlplane provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(controlPlaneProviderComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Deleting config maps with controlplane provider manifests")
		for _, cm := range configMaps {
			Expect(bootstrapCluster.Delete(ctx, &cm)).To(Succeed())
		}

		By("Deleting capkcp-system namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: capkcpSystemNamespace,
			},
		}
		Expect(bootstrapCluster.Delete(ctx, namespace)).To(Succeed())
	})
})
