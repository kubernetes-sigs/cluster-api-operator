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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "sigs.k8s.io/cluster-api-operator/test/framework"
)

var _ = Describe("Create a proper set of manifests when using helm charts", func() {
	BeforeEach(func() {
		// Ensure that there are no Cluster API CRDs from previous tests
		deleteClusterAPICRDs(helmClusterProxy)
	})

	It("should deploy a quick-start cluster-api-operator-providers chart", func() {
		clusterProxy := helmClusterProxy.GetClient()

		fullHelmChart := &HelmChart{
			BinaryPath:      helmBinaryPath,
			Path:            chartPath,
			Name:            capiOperatorRelease,
			Kubeconfig:      helmClusterProxy.GetKubeconfigPath(),
			Wait:            true,
			Output:          Full,
			AdditionalFlags: Flags("--create-namespace", "--namespace", operatorNamespace),
		}

		defer func() {
			fullHelmChart.Commands = Commands(Uninstall)
			fullHelmChart.AdditionalFlags = Flags("--namespace", operatorNamespace)
			_, err := fullHelmChart.Run(nil)
			Expect(err).ToNot(HaveOccurred())

			err = clusterProxy.DeleteAllOf(ctx, &apiextensionsv1.CustomResourceDefinition{}, client.MatchingLabels{
				clusterctlv1.ClusterctlCoreLabel: capiOperatorRelease,
			})
			Expect(err).ToNot(HaveOccurred())
		}()

		_, err := fullHelmChart.Run(nil)
		Expect(err).ToNot(HaveOccurred())

		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
		}
		Expect(clusterProxy.Create(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     clusterProxy,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(clusterProxy).ToSatisfy(
			HaveStatusConditionsTrue(coreProvider, operatorv1.ProviderInstalledCondition),
		), e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(clusterProxy).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, ptr.To(coreProvider.Spec.Version))
		}), e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		bootstrapProvider := &operatorv1.BootstrapProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderDeploymentName,
			Namespace: operatorNamespace,
		}}

		Expect(clusterProxy.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     clusterProxy,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(clusterProxy).ToSatisfy(
			HaveStatusConditionsTrue(bootstrapProvider, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(bootstrapProvider).In(clusterProxy).ToSatisfy(func() bool {
			return ptr.Equal(bootstrapProvider.Status.InstalledVersion, &bootstrapProvider.Spec.Version)
		}), e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)
		Expect(clusterProxy.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(clusterProxy),
			e2eConfig.GetIntervals(helmClusterProxy.GetName(), "wait-controllers")...)

		Expect(clusterProxy.Delete(ctx, coreProvider)).To(Succeed())
	})

	It("should deploy default manifest set for quick-start process", func() {
		fullRun := &HelmChart{
			BinaryPath: helmChart.BinaryPath,
			Path:       helmChart.Path,
			Name:       helmChart.Name,
			Kubeconfig: helmChart.Kubeconfig,
			DryRun:     helmChart.DryRun,
			Output:     Manifests,
		}
		fullRun.Output = Manifests
		manifests, err := fullRun.Run(nil)
		Expect(err).ToNot(HaveOccurred())
		fullChartInstall, err := os.ReadFile(filepath.Join(customManifestsFolder, "full-chart-install.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(fullChartInstall)))
	})

	It("should not deploy providers when none specified", func() {
		manifests, err := helmChart.Run(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(BeEmpty())
	})

	It("should deploy all providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":               "test-secret-name",
			"configSecret.namespace":          "test-secret-namespace",
			"core.cluster-api.namespace":      "capi-custom-ns",
			"core.cluster-api.version":        "v1.7.7",
			"controlPlane.kubeadm.namespace":  "kubeadm-control-plane-custom-ns",
			"controlPlane.kubeadm.version":    "v1.7.7",
			"bootstrap.kubeadm.namespace":     "kubeadm-bootstrap-custom-ns",
			"bootstrap.kubeadm.version":       "v1.7.7",
			"infrastructure.docker.namespace": "capd-custom-ns",
			"infrastructure.docker.version":   "v1.7.7",
			"ipam.in-cluster.namespace":       "in-cluster-custom-ns",
			"ipam.in-cluster.version":         "v1.0.0",
			"addon.helm.namespace":            "helm-custom-ns",
			"addon.helm.version":              "v0.2.6",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy all providers with custom versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":             "test-secret-name",
			"configSecret.namespace":        "test-secret-namespace",
			"core.cluster-api.version":      "v1.7.7",
			"controlPlane.kubeadm.version":  "v1.7.7",
			"bootstrap.kubeadm.version":     "v1.7.7",
			"infrastructure.docker.version": "v1.7.7",
			"ipam.in-cluster.version":       "v1.0.0",
			"addon.helm.version":            "v0.2.6",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-custom-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy all providers with latest version", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":             "test-secret-name",
			"configSecret.namespace":        "test-secret-namespace",
			"core.cluster-api.enabled":      "true",
			"controlPlane.kubeadm.enabled":  "true",
			"bootstrap.kubeadm.enabled":     "true",
			"infrastructure.docker.enabled": "true",
			"ipam.in-cluster.enabled":       "true",
			"addon.helm.enabled":            "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-latest-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core, bootstrap, control plane when only infra is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":             "test-secret-name",
			"configSecret.namespace":        "test-secret-namespace",
			"infrastructure.docker.enabled": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-infra.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only bootstrap is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":         "test-secret-name",
			"configSecret.namespace":    "test-secret-namespace",
			"bootstrap.kubeadm.enabled": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-bootstrap.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only control plane is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":            "test-secret-name",
			"configSecret.namespace":       "test-secret-namespace",
			"controlPlane.kubeadm.enabled": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-control-plane.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only ipam is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":       "test-secret-name",
			"configSecret.namespace":  "test-secret-namespace",
			"ipam.in-cluster.enabled": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-ipam.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core, bootstrap, control plane when only infra and ipam is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":             "test-secret-name",
			"configSecret.namespace":        "test-secret-namespace",
			"infrastructure.docker.enabled": "true",
			"ipam.in-cluster.enabled":       "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-infra-and-ipam.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple infra providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":               "test-secret-name",
			"configSecret.namespace":          "test-secret-namespace",
			"infrastructure.docker.namespace": "capd-custom-ns",
			"infrastructure.docker.version":   "v1.7.7",
			"infrastructure.azure.namespace":  "capz-custom-ns",
			"infrastructure.azure.version":    "v1.10.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-infra-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple control plane providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":              "test-secret-name",
			"configSecret.namespace":         "test-secret-namespace",
			"controlPlane.kubeadm.namespace": "kubeadm-control-plane-custom-ns",
			"controlPlane.kubeadm.version":   "v1.7.7",
			"controlPlane.rke2.namespace":    "rke2-control-plane-custom-ns",
			"controlPlane.rke2.version":      "v0.8.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-control-plane-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple bootstrap providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":           "test-secret-name",
			"configSecret.namespace":      "test-secret-namespace",
			"bootstrap.kubeadm.namespace": "kubeadm-bootstrap-custom-ns",
			"bootstrap.kubeadm.version":   "v1.7.7",
			"bootstrap.rke2.namespace":    "rke2-bootstrap-custom-ns",
			"bootstrap.rke2.version":      "v0.8.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-bootstrap-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only addon is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"addon.helm.enabled":     "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-addon.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core, bootstrap, control plane when only infra and addon is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":             "test-secret-name",
			"configSecret.namespace":        "test-secret-namespace",
			"infrastructure.docker.enabled": "true",
			"addon.helm.enabled":            "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-infra-and-addon.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
	It("should deploy core and infra with feature gates enabled", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":                         "aws-variables",
			"configSecret.namespace":                    "default",
			"infrastructure.aws.version":                "v2.4.0",
			"ipam.in-cluster.enabled":                   "true",
			"addon.helm.enabled":                        "true",
			"image.manager.tag":                         "v0.9.1",
			"core.cluster-api.version":                  "v1.6.2",
			"manager.featureGates.core.ClusterTopology": "true",
			"manager.featureGates.core.MachinePool":     "true",
			"manager.featureGates.aws.ClusterTopology":  "true",
			"manager.featureGates.aws.MachinePool":      "true",
			"manager.featureGates.aws.EKSEnableIAM":     "true",
			"manager.featureGates.aws.EKSAllowAddRoles": "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "feature-gates.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
	It("should deploy all providers with manager defined but no feature gates enabled", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":            "test-secret-name",
			"configSecret.namespace":       "test-secret-namespace",
			"core.cluster-api.enabled":     "true",
			"infrastructure.azure.enabled": "true",
			"ipam.in-cluster.enabled":      "true",
			"addon.helm.enabled":           "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-manager-defined-no-feature-gates.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
	It("should deploy all providers when manager is defined but another infrastructure spec field is defined", func() {
		manifests, err := helmChart.Run(map[string]string{
			"core.cluster-api.enabled":                  "true",
			"controlPlane.kubeadm.enabled":              "true",
			"bootstrap.kubeadm.enabled":                 "true",
			"infrastructure.docker.enabled":             "true",
			"ipam.in-cluster.enabled":                   "true",
			"addon.helm.enabled":                        "true",
			"manager.featureGates.core.ClusterTopology": "true",
			"manager.featureGates.core.MachinePool":     "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "manager-defined-missing-other-infra-spec.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
	It("should deploy kubeadm control plane with manager specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"core.cluster-api.enabled":                     "true",
			"controlPlane.kubeadm.enabled":                 "true",
			"bootstrap.kubeadm.enabled":                    "true",
			"infrastructure.docker.enabled":                "true",
			"ipam.in-cluster.enabled":                      "true",
			"addon.helm.enabled":                           "true",
			"manager.featureGates.kubeadm.ClusterTopology": "true",
			"manager.featureGates.kubeadm.MachinePool":     "true",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "kubeadm-manager-defined.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
})
