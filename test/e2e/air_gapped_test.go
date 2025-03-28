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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	. "sigs.k8s.io/cluster-api-operator/test/framework"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var namespaces = []string{cabpkSystemNamespace, cacpkSystemNamespace, capiSystemNamespace}

var _ = Describe("Install ControlPlane, Core, Bootstrap providers in an air-gapped environment", Ordered, func() {
	var (
		configMaps       []corev1.ConfigMap
		bootstrapCluster client.Client
		coreProvider     *operatorv1.CoreProvider
	)

	BeforeAll(func() {
		bootstrapCluster = bootstrapClusterProxy.GetClient()
		coreProvider = &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: capiSystemNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								operatorv1.ConfigMapNameLabel:        coreProviderName,
								operatorv1.ConfigMapTypeLabel:        "core",
								operatorv1.ConfigMapVersionLabelName: "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
				},
			},
		}

		// Ensure that there are no Cluster API installed
		deleteClusterAPICRDs(bootstrapClusterProxy)

		By("should successfully create ConfigMaps with ControlPlane, Core, and Bootstrap provider manifests")
		configMapFiles := []string{
			"core-cluster-api-v1.7.7.yaml",
			"core-cluster-api-v1.8.0.yaml",
			"bootstrap-kubeadm-v1.7.7.yaml",
			"bootstrap-kubeadm-v1.8.0.yaml",
			"controlplane-kubeadm-v1.7.7.yaml",
			"controlplane-kubeadm-v1.8.0.yaml",
		}

		for _, fileName := range configMapFiles {
			providerComponents, err := os.ReadFile(filepath.Join(customManifestsFolder, fileName))
			Expect(err).ToNot(HaveOccurred(), "Failed to read the provider manifests file")

			var configMap corev1.ConfigMap

			Expect(yaml.Unmarshal(providerComponents, &configMap)).To(Succeed())

			configMaps = append(configMaps, configMap)
		}

		By("Creating provider namespaces")
		for _, namespaceName := range namespaces {
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

		By("Installing Core provider")
		Expect(bootstrapCluster.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for Core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(coreProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))).To(BeTrue())

		By("Waiting for the Core provider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: capiSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		DeferCleanup(func() {
			By("Deleting Core provider")
			Expect(bootstrapCluster.Delete(ctx, coreProvider)).To(Succeed())

			By("Waiting for the Core provider Deployment to be deleted")
			WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderDeploymentName,
				Namespace: capiSystemNamespace,
			}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

			By("Waiting for the Core provider object to be deleted")
			WaitForDelete(
				ctx, For(coreProvider).In(bootstrapCluster),
				e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

			By("Deleting ConfigMaps with ControlPlane, Core, and Bootstrap provider manifests")
			for _, cm := range configMaps {
				Expect(bootstrapCluster.Delete(ctx, &cm)).To(Succeed())
			}

			By("Deleting provider namespaces")
			for _, namespaceName := range namespaces {
				namespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespaceName,
					},
				}
				Expect(bootstrapCluster.Delete(ctx, namespace)).To(Succeed())
			}
		})
	})

	It("should successfully create, upgrade (v1.7.7 -> v1.8.0) and delete a BootstrapProvider from a ConfigMap", func() {
		bootstrapProvider := &operatorv1.BootstrapProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customProviderName,
				Namespace: cabpkSystemNamespace,
			},
			Spec: operatorv1.BootstrapProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								operatorv1.ConfigMapNameLabel:        "kubeadm",
								operatorv1.ConfigMapTypeLabel:        "bootstrap",
								operatorv1.ConfigMapVersionLabelName: "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for BootstrapProvider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(bootstrapProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the BootstrapProvider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: bootstrapProviderDeploymentName, Namespace: cabpkSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(bootstrapProvider.Status.InstalledVersion, ptr.To(bootstrapProvider.Spec.Version))).To(BeTrue())

		By("Updating the BootstrapProvider to new Custer API version")
		patch := client.MergeFrom(bootstrapProvider.DeepCopy())
		bootstrapProvider.Spec.Version = nextCAPIVersion
		bootstrapProvider.Spec.FetchConfig.Selector.MatchLabels[operatorv1.ConfigMapVersionLabelName] = nextCAPIVersion
		Expect(bootstrapCluster.Patch(ctx, bootstrapProvider, patch)).To(Succeed())

		By("Waiting for BootstrapProvider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(bootstrapProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, operatorv1.ProviderUpgradedCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the BootstrapProvider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: bootstrapProviderDeploymentName, Namespace: cabpkSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(bootstrapProvider.Status.InstalledVersion, ptr.To(bootstrapProvider.Spec.Version))).To(BeTrue())

		By("Deleting BootstrapProvider provider")
		Expect(bootstrapCluster.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the BootstrapProvider Deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderDeploymentName,
			Namespace: cabpkSystemNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the BootstrapProvider object to be deleted")
		WaitForDelete(
			ctx, For(bootstrapProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create, upgrade (v1.7.7 -> v1.8.0) and delete a ControlPlaneProvider from a ConfigMap", func() {
		controlPlaneProvider := &operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      customProviderName,
				Namespace: cacpkSystemNamespace,
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								operatorv1.ConfigMapNameLabel:        "kubeadm",
								operatorv1.ConfigMapTypeLabel:        "controlplane",
								operatorv1.ConfigMapVersionLabelName: "v1.7.7",
							},
						},
					},
					Version: previousCAPIVersion,
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, controlPlaneProvider)).To(Succeed())

		By("Waiting for ControlPlaneProvider to be ready")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(controlPlaneProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the ControlPlaneProvider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: cpProviderDeploymentName, Namespace: cacpkSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(controlPlaneProvider.Status.InstalledVersion, ptr.To(controlPlaneProvider.Spec.Version))).To(BeTrue())

		By("Updating the ControlPlaneProvider to new Custer API version")
		patch := client.MergeFrom(controlPlaneProvider.DeepCopy())
		controlPlaneProvider.Spec.Version = nextCAPIVersion
		controlPlaneProvider.Spec.FetchConfig.Selector.MatchLabels[operatorv1.ConfigMapVersionLabelName] = nextCAPIVersion
		Expect(bootstrapCluster.Patch(ctx, controlPlaneProvider, patch)).To(Succeed())

		By("Waiting for ControlPlaneProvider to be ready")
		WaitFor(ctx, For(controlPlaneProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(controlPlaneProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, operatorv1.ProviderUpgradedCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the ControlPlaneProvider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: cpProviderDeploymentName, Namespace: cacpkSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(controlPlaneProvider.Status.InstalledVersion, ptr.To(controlPlaneProvider.Spec.Version))).To(BeTrue())

		By("Deleting ControlPlaneProvider provider")
		Expect(bootstrapCluster.Delete(ctx, controlPlaneProvider)).To(Succeed())

		By("Waiting for the ControlPlaneProvider Deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderDeploymentName,
			Namespace: cacpkSystemNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the ControlPlaneProvider object to be deleted")
		WaitForDelete(
			ctx, For(controlPlaneProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade CoreProvider (v1.7.7 -> v1.8.0)", func() {
		Expect(bootstrapCluster.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())

		By("Updating the CoreProvider to new Custer API version")
		patch := client.MergeFrom(coreProvider.DeepCopy())
		coreProvider.Spec.Version = nextCAPIVersion
		coreProvider.Spec.FetchConfig.Selector.MatchLabels[operatorv1.ConfigMapVersionLabelName] = nextCAPIVersion
		Expect(bootstrapCluster.Patch(ctx, coreProvider, patch)).To(Succeed())

		By("Waiting for CoreProvider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusConditionsTrue(coreProvider, operatorv1.PreflightCheckCondition, operatorv1.ProviderInstalledCondition, operatorv1.ProviderUpgradedCondition, clusterv1.ReadyCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the CoreProvider Deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapClusterProxy.GetClient(),
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: capiSystemNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Validating that status.InstalledVersion is set")
		Expect(ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))).To(BeTrue())
	})
})
