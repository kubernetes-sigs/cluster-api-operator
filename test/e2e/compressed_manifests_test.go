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
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api/test/framework"

	"k8s.io/utils/ptr"
	. "sigs.k8s.io/cluster-api-operator/test/framework"
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
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
		}
		Expect(bootstrapCluster.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
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

	It("should successfully create and delete an InfrastructureProvider for OCI", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
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

		Expect(bootstrapCluster.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider to be ready")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&infraProvider.Status.Conditions, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(infraProvider.Status.InstalledVersion, ptr.To(infraProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Ensure that the created config map has correct annotation")
		cmName := fmt.Sprintf("infrastructure-%s-%s", ociInfrastructureProviderName, ociInfrastructureProviderVersion)
		key := client.ObjectKey{Namespace: operatorNamespace, Name: cmName}

		// Save config map contents to be used later.
		Expect(bootstrapCluster.Get(ctx, key, ociInfrastructureConfigMap)).To(Succeed())

		Expect(ociInfrastructureConfigMap.GetAnnotations()[compressedAnnotation]).To(Equal("true"))

		Expect(ociInfrastructureConfigMap.BinaryData[componentsConfigMapKey]).ToNot(BeEmpty())

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      ociInfrastructureProviderDeploymentName,
		}}

		By("Waiting for the infrastructure provider deployment to be created")
		WaitFor(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the configmap to be deleted")
		WaitForDelete(ctx, For(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      cmName,
		}}).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an InfrastructureProvider for OCI with custom name from a pre-created ConfigMap", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
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
			}}

		// Re-use configmap created on the previous step.
		ociInfrastructureConfigMap.ObjectMeta.UID = ""
		ociInfrastructureConfigMap.ObjectMeta.ResourceVersion = ""
		ociInfrastructureConfigMap.ObjectMeta.CreationTimestamp = metav1.Time{}
		ociInfrastructureConfigMap.ObjectMeta.OwnerReferences = nil
		Expect(bootstrapCluster.Create(ctx, ociInfrastructureConfigMap)).To(Succeed())

		Expect(bootstrapCluster.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider to be ready")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&infraProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.IntalledVersion to be set")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(infraProvider.Status.InstalledVersion, ptr.To(infraProvider.Spec.Version))
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Ensure that the created config map has correct annotation")
		cm := &corev1.ConfigMap{}
		cmName := fmt.Sprintf("infrastructure-%s-%s", ociInfrastructureProviderName, ociInfrastructureProviderVersion)
		key := client.ObjectKey{Namespace: operatorNamespace, Name: cmName}
		Expect(bootstrapCluster.Get(ctx, key, cm)).To(Succeed())

		Expect(cm.GetAnnotations()[compressedAnnotation]).To(Equal("true"))

		Expect(cm.BinaryData[componentsConfigMapKey]).ToNot(BeEmpty())

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      ociInfrastructureProviderDeploymentName,
		}}

		By("Waiting for the infrastructure provider deployment to be created")
		WaitFor(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, For(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderDeploymentName,
			Namespace: operatorNamespace,
		}}).In(bootstrapCluster), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(ctx, For(coreProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})
})
