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
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api/test/framework"

	. "sigs.k8s.io/cluster-api-operator/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Create, upgrade, downgrade and delete providers with minimal specified configuration", func() {
	It("should successfully create a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()

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

		Expect(bootstrapCluster.Create(ctx, additionalManifests)).To(Succeed())

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

		Expect(bootstrapCluster.Create(ctx, coreProvider)).To(Succeed())
		Expect(bootstrapCluster.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())
		Expect(coreProvider.Spec.AdditionalManifestsRef).ToNot(BeNil())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, &coreProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Checking if additional manifests are applied")
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config-map",
			Namespace: operatorNamespace,
		}}
		WaitFor(ctx, For(cm).In(bootstrapCluster).ToSatisfy(func() bool {
			value, ok := cm.Data["test"]
			return ok && value == "test"
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete a BootstrapProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderDeploymentName,
			Namespace: operatorNamespace,
		}}

		Expect(bootstrapCluster.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&bootstrapProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(bootstrapProvider.Status.InstalledVersion, &bootstrapProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
		Expect(bootstrapCluster.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete a ControlPlaneProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		cpProvider := &operatorv1.ControlPlaneProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderDeploymentName,
			Namespace: operatorNamespace,
		}}

		Expect(bootstrapCluster.Create(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the control plane provider to be ready")
		WaitFor(ctx, For(cpProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&cpProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(cpProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(cpProvider.Status.InstalledVersion, &cpProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an InfrastructureProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      infraProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      infraProviderDeploymentName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the infrastructure provider to be ready")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&infraProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(infraProvider.Status.InstalledVersion, &infraProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an AddonProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		addonProvider := &operatorv1.AddonProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      addonProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      addonProviderDeploymentName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, addonProvider)).To(Succeed())

		By("Waiting for the addon provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the addon provider to be ready")
		WaitFor(ctx, For(addonProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&addonProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(addonProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(addonProvider.Status.InstalledVersion, &addonProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, addonProvider)).To(Succeed())

		By("Waiting for the addon provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully downgrade a CoreProvider (latest -> v1.4.2)", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{}
		key := client.ObjectKey{Namespace: operatorNamespace, Name: coreProviderName}
		Expect(bootstrapCluster.Get(ctx, key, coreProvider)).To(Succeed())

		coreProvider.Spec.Version = previousCAPIVersion

		Expect(bootstrapCluster.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready and status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider to have status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, &coreProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade a CoreProvider (v1.4.2 -> latest)", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())

		coreProvider.Spec.Version = ""

		Expect(bootstrapCluster.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provide status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, &coreProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      coreProviderDeploymentName,
		}}

		Expect(bootstrapCluster.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(ctx, For(coreProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})
})
