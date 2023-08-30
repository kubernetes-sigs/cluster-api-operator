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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "sigs.k8s.io/cluster-api-operator/test/framework"
)

var _ = Describe("Create a proper set of manifests when using helm charts", func() {
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
		Expect(manifests).To(Equal(string(fullChartInstall)))
	})

	It("should not deploy providers when none specified", func() {
		manifests, err := helmChart.Run(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(BeEmpty())
	})

	It("should deploy all providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"core":                   "capi-custom-ns:cluster-api:v1.4.2",
			"controlPlane":           "kubeadm-control-plane-custom-ns:kubeadm:v1.4.2",
			"bootstrap":              "kubeadm-bootstrap-custom-ns:kubeadm:v1.4.2",
			"infrastructure":         "capd-custom-ns:docker:v1.4.2",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy all providers with custom versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"core":                   "cluster-api:v1.4.2",
			"controlPlane":           "kubeadm:v1.4.2",
			"bootstrap":              "kubeadm:v1.4.2",
			"infrastructure":         "docker:v1.4.2",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-custom-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy all providers with latest version", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"core":                   "cluster-api",
			"controlPlane":           "kubeadm",
			"bootstrap":              "kubeadm",
			"infrastructure":         "docker",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "all-providers-latest-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core, bootstrap, control plane when only infra is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"infrastructure":         "docker",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-infra.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only bootstrap is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"bootstrap":              "kubeadm",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-bootstrap.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy core when only control plane is specified", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"controlPlane":           "kubeadm",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "only-control-plane.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple infra providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"infrastructure":         "capd-custom-ns:docker:v1.4.2;capz-custom-ns:azure:v1.10.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-infra-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple control plane providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"controlPlane":           "kubeadm-control-plane-custom-ns:kubeadm:v1.4.2;rke2-control-plane-custom-ns:rke2:v0.3.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-control-plane-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})

	It("should deploy multiple bootstrap providers with custom namespace and versions", func() {
		manifests, err := helmChart.Run(map[string]string{
			"configSecret.name":      "test-secret-name",
			"configSecret.namespace": "test-secret-namespace",
			"bootstrap":              "kubeadm-bootstrap-custom-ns:kubeadm:v1.4.2;rke2-bootstrap-custom-ns:rke2:v0.3.0",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).ToNot(BeEmpty())
		expectedManifests, err := os.ReadFile(filepath.Join(customManifestsFolder, "multiple-bootstrap-custom-ns-versions.yaml"))
		Expect(err).ToNot(HaveOccurred())
		Expect(manifests).To(Equal(string(expectedManifests)))
	})
})
