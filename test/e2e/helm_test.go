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

	. "sigs.k8s.io/cluster-api-operator/test/framework"
)

var _ = Describe("Create a proper set of manifests when using helm charts", func() {
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
