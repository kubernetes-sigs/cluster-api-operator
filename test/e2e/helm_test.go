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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	. "sigs.k8s.io/cluster-api-operator/test/framework"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	It("should render operator chart manifests matching expected output", func() {
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
		Expect(manifests).To(Equal(strings.TrimSpace(string(fullChartInstall))))
	})
})
