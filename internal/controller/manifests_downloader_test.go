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
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/yamlprocessor"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

func TestManifestsDownloader(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	p := prepareReconciler(ctx, g, clusterctlv1.CoreProviderType, "",
		&operatorv1.CoreProvider{
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
	)

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
	p := prepareReconciler(ctx, g, clusterctlv1.CoreProviderType, "https://github.com/kubernetes-sigs/cluster-api/releases/v1.4.3/core-components.yaml",
		&operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-api",
				Namespace: testNamespaceName,
			},
			Spec: operatorv1.CoreProviderSpec{},
		},
	)

	_, err := p.load(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.fetch(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(p.components.Images()).To(HaveExactElements([]string{"registry.k8s.io/cluster-api/cluster-api-controller:v1.4.3"}))
	g.Expect(p.components.Version()).To(Equal("v1.4.3"))
}

func TestFetchProviderName(t *testing.T) {
	testCases := []struct {
		name         string
		providerType clusterctlv1.ProviderType
		provider     genericprovider.GenericProvider
		variables    map[string]string
		url          string
		expected     string
	}{
		{
			name:         "Helm addon provider manifest",
			providerType: clusterctlv1.AddonProviderType,
			provider: &operatorv1.AddonProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "addon-helm",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.AddonProviderSpec{},
			},
			url:      "https://github.com/kubernetes-sigs/cluster-api-addon-provider-helm/releases/v0.2.6/addon-components.yaml",
			expected: "helm",
		},
		{
			name:         "vSphere infrastructure provider manifest",
			providerType: clusterctlv1.InfrastructureProviderType,
			provider: &operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infrastructure-vsphere",
					Namespace: testNamespaceName,
				},
				Spec: operatorv1.InfrastructureProviderSpec{},
			},
			variables: map[string]string{
				"VSPHERE_PASSWORD": "password",
				"VSPHERE_USERNAME": "username",
			},
			url:      "https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/releases/v1.12.0/infrastructure-components.yaml",
			expected: "vsphere",
		},
	}
	g := NewWithT(t)

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := prepareReconciler(ctx, g, tc.providerType, tc.url, tc.provider)
			_, err := p.load(ctx)
			g.Expect(err).ToNot(HaveOccurred())

			componentsFile, err := p.repo.GetFile(ctx, p.options.Version, p.repo.ComponentsPath())
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(componentsFile).ToNot(BeNil())

			for k, v := range tc.variables {
				p.configClient.Variables().Set(k, v)
			}

			input := repository.ComponentsInput{
				Provider:     p.providerConfig,
				ConfigClient: p.configClient,
				Processor:    yamlprocessor.NewSimpleProcessor(),
				RawYaml:      componentsFile,
				Options:      p.options,
			}

			providerName, err := fetchProviderName(input)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(providerName).To(Equal(tc.expected))
		})
	}
}

func prepareReconciler(ctx context.Context, g *WithT, providerType clusterctlv1.ProviderType, url string, provider genericprovider.GenericProvider) *phaseReconciler {
	var overridesClient configclient.Client

	if url != "" {
		reader := configclient.NewMemoryReader()
		_, err := reader.AddProvider(provider.GetName(), providerType, url)
		g.Expect(err).ToNot(HaveOccurred())

		overridesClient, err = configclient.New(ctx, "", configclient.InjectReader(reader))
		g.Expect(err).ToNot(HaveOccurred())
	}

	p := &phaseReconciler{
		ctrlClient:      fake.NewClientBuilder().WithObjects().Build(),
		provider:        provider,
		overridesClient: overridesClient,
	}

	_, err := p.initializePhaseReconciler(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.downloadManifests(ctx)
	g.Expect(err).ToNot(HaveOccurred())

	return p
}
