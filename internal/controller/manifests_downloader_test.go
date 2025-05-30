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

package controller

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

func TestManifestsDownloader(t *testing.T) {
	g := NewWithT(t)

	ctx := context.Background()

	fakeclient := fake.NewClientBuilder().WithObjects().Build()

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider: &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-api",
				Namespace: testNamespaceName,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: "v1.4.3",
				},
			},
		},
	}

	_, err := p.InitializePhaseReconciler(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.DownloadManifests(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	// Ensure that config map was created
	labelSelector := metav1.LabelSelector{
		MatchLabels: p.prepareConfigMapLabels(),
	}

	exists, err := p.checkConfigMapExists(ctx, labelSelector, p.provider.GetNamespace())
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(exists).To(BeTrue())
}

func TestProviderDownloadWithOverrides(t *testing.T) {
	g := NewWithT(t)

	ctx := context.Background()

	fakeclient := fake.NewClientBuilder().WithObjects().Build()

	reader := configclient.NewMemoryReader()
	_, err := reader.AddProvider("cluster-api", clusterctlv1.CoreProviderType, "https://github.com/kubernetes-sigs/cluster-api/releases/v1.4.3/core-components.yaml")
	g.Expect(err).ToNot(HaveOccurred())

	overridesClient, err := configclient.New(ctx, "", configclient.InjectReader(reader))
	g.Expect(err).ToNot(HaveOccurred())

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider: &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-api",
				Namespace: testNamespaceName,
			},
			Spec: operatorv1.CoreProviderSpec{},
		},
		overridesClient: overridesClient,
	}

	_, err = p.InitializePhaseReconciler(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.DownloadManifests(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.Load(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.Fetch(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(p.components.Images()).To(HaveExactElements([]string{"registry.k8s.io/cluster-api/cluster-api-controller:v1.4.3"}))
	g.Expect(p.components.Version()).To(Equal("v1.4.3"))
}
