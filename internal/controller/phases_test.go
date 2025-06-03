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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
)

func TestSecretReader(t *testing.T) {
	g := NewWithT(t)

	fakeclient := fake.NewClientBuilder().WithObjects().Build()

	secretName := "test-secret"
	secretNamespace := "test-secret-namespace"

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider: &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-api",
				Namespace: testNamespaceName,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "CoreProvider",
				APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					ConfigSecret: &operatorv1.SecretReference{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					FetchConfig: &operatorv1.FetchConfiguration{
						URL: "https://example.com",
					},
				},
			},
		},
	}

	testKey1 := "test-key1"
	testValue1 := "test-value1"
	testKey2 := "test-key2"
	testValue2 := "test-value2"
	testKey3 := "test-key3"
	testValue3 := "test-value3"

	g.Expect(fakeclient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Data: map[string][]byte{
			testKey1: []byte(testValue1),
			testKey2: []byte(testValue2),
		},
	})).To(Succeed())

	configreader, err := p.secretReader(context.TODO(), configclient.NewProvider(testKey3, testValue3, clusterctlv1.CoreProviderType))
	g.Expect(err).ToNot(HaveOccurred())

	expectedValue1, err := configreader.Get(testKey1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(expectedValue1).To(Equal(testValue1))

	expectedValue2, err := configreader.Get(testKey2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(expectedValue2).To(Equal(testValue2))

	expectedProviderData, err := configreader.Get("providers")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(expectedProviderData).To(Equal(`- name: test-key3
  type: CoreProvider
  url: test-value3
- name: cluster-api
  type: CoreProvider
  url: https://example.com
`))
}

func TestConfigmapRepository(t *testing.T) {
	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws",
			Namespace: "ns1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "InfrastructureProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"provider-components": "aws"},
					},
				},
			},
		},
	}
	metadata := `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 0
	minor: 4
	contract: v1alpha4
  - major: 0
	minor: 3
	contract: v1alpha3`

	components := `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    cluster.x-k8s.io/provider: cluster-api
    control-plane: controller-manager
  name: capi-system
---
apiVersion: v1
kind: Namespace
metadata:
  labels:
    cluster.x-k8s.io/provider: cluster-api
    control-plane: controller-manager
  name: capi-webhook-system
---`

	additionalManifests := `
apiVersion: v1
kind: Namespace
metadata:
  name: some-other-namespace
---
`
	tests := []struct {
		name                string
		configMaps          []corev1.ConfigMap
		additionalManifests string
		want                repository.Repository
		wantErr             string
		wantDefaultVersion  string
	}{
		{
			name:    "missing configmaps",
			wantErr: "no ConfigMaps found with selector &LabelSelector{MatchLabels:map[string]string{provider-components: aws,},MatchExpressions:[]LabelSelectorRequirement{},}",
		},
		{
			name: "configmap with missing metadata",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.3",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{"components": components},
				},
			},
			wantErr: "ConfigMap ns1/v1.2.3 has no metadata",
		},
		{
			name: "configmap with missing components",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.3",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata": metadata,
					},
				},
			},
			wantErr: "ConfigMap ns1/v1.2.3 Data has no components",
		},
		{
			name: "configmap with invalid version in the name",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "not-a-version",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata": metadata,
					},
				},
			},
			wantErr: "ConfigMap ns1/not-a-version has invalid version:not-a-version (from the Name)",
		},
		{
			name: "configmap with invalid version in the Label",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "not-a-version",
						Namespace: "ns1",
						Labels: map[string]string{
							"provider-components":               "aws",
							"provider.cluster.x-k8s.io/version": "also-not-a-label",
						},
					},
					Data: map[string]string{
						"metadata": metadata,
					},
				},
			},
			wantErr: "ConfigMap ns1/not-a-version has invalid version:also-not-a-label (from the Label provider.cluster.x-k8s.io/version)",
		},
		{
			name: "one correct configmap",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.3",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
			},
			wantDefaultVersion: "v1.2.3",
		},
		{
			name: "one correct configmap with label version",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-provider",
						Namespace: "ns1",
						Labels: map[string]string{
							"provider-components":               "aws",
							"provider.cluster.x-k8s.io/version": "v1.2.3",
						},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
			},
			wantDefaultVersion: "v1.2.3",
		},
		{
			name: "three correct configmaps",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.3",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.7",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.4",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
			},
			wantDefaultVersion: "v1.2.3",
		},
		{
			name: "with additional manifests",
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v1.2.3",
						Namespace: "ns1",
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{
						"metadata":   metadata,
						"components": components,
					},
				},
			},
			additionalManifests: additionalManifests,
			wantDefaultVersion:  "v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			fakeclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(provider).Build()
			p := &PhaseReconciler{
				ctrlClient: fakeclient,
				provider:   provider,
			}

			for i := range tt.configMaps {
				g.Expect(fakeclient.Create(ctx, &tt.configMaps[i])).To(Succeed())
			}

			got, err := p.configmapRepository(context.TODO(), p.provider.GetSpec().FetchConfig.Selector, InNamespace("ns1"), WithAdditionalManifests(tt.additionalManifests))

			if len(tt.wantErr) > 0 {
				g.Expect(err).Should(MatchError(tt.wantErr))
				return
			}

			g.Expect(err).To(Succeed())
			gotComponents, err := got.GetFile(ctx, got.DefaultVersion(), got.ComponentsPath())
			g.Expect(err).To(Succeed())

			if tt.additionalManifests != "" {
				g.Expect(string(gotComponents)).To(Equal(components + "\n---\n" + additionalManifests))
			} else {
				g.Expect(string(gotComponents)).To(Equal(components))
			}

			gotMetadata, err := got.GetFile(ctx, got.DefaultVersion(), "metadata.yaml")
			g.Expect(err).To(Succeed())
			g.Expect(string(gotMetadata)).To(Equal(metadata))

			g.Expect(got.DefaultVersion()).To(Equal(tt.wantDefaultVersion))
		})
	}
}

func TestRepositoryProxy(t *testing.T) {
	coreProvider := configclient.NewProvider("cluster-api", "https://github.com/kubernetes-sigs/cluster-api/releases/latest/core-components.yaml", clusterctlv1.CoreProviderType)
	awsProvider := configclient.NewProvider("aws", "https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/v1.4.1/infrastructure-components.yaml", clusterctlv1.InfrastructureProviderType)

	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws",
			Namespace: "ns1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "InfrastructureProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v2.3.5",
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"provider-components": "aws"},
					},
				},
			},
		},
	}

	core := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "default",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}

	awsMetadata := `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 2
    minor: 4
    contract: v1beta1
  - major: 2
    minor: 3
    contract: v1beta1`

	awsMetaReleaseSeries := []clusterctlv1.ReleaseSeries{
		{
			Major:    2,
			Minor:    4,
			Contract: "v1beta1",
		}, {
			Major:    2,
			Minor:    3,
			Contract: "v1beta1",
		},
	}

	metadata := `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 0
    minor: 4
    contract: v1alpha4
  - major: 0
    minor: 3
    contract: v1alpha3`

	metaReleaseSeries := []clusterctlv1.ReleaseSeries{{
		Major:    0,
		Minor:    4,
		Contract: "v1alpha4",
	}, {
		Major:    0,
		Minor:    3,
		Contract: "v1alpha3",
	}}

	tests := []struct {
		name               string
		configMaps         []corev1.ConfigMap
		genericProviders   []client.Object
		provider           configclient.Provider
		defaultRepository  bool
		wantMetadataSeries []clusterctlv1.ReleaseSeries
		wantErr            string
		metadataErr        string
		wantDefaultVersion string
	}{
		{
			name:               "missing configmaps",
			provider:           coreProvider,
			wantDefaultVersion: testCurrentVersion,
			genericProviders:   []client.Object{core, provider},
			metadataErr:        "failed to read \"metadata.yaml\" from the repository for provider \"cluster-api\": unable to get files for version v0.4.2",
		},
		{
			name:               "correct configmap with data",
			genericProviders:   []client.Object{core, provider},
			provider:           coreProvider,
			defaultRepository:  true,
			wantDefaultVersion: testCurrentVersion,
			wantMetadataSeries: metaReleaseSeries,
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      testCurrentVersion,
						Namespace: "default",
						Labels: map[string]string{
							operatorv1.ConfigMapVersionLabelName: testCurrentVersion,
							operatorv1.ConfigMapTypeLabel:        core.GetType(),
							operatorv1.ConfigMapNameLabel:        core.GetName(),
							operatorManagedLabel:                 "true",
						},
					},
					Data: map[string]string{"metadata": metadata, "components": ""},
				},
			},
		},
		{
			name:             "upgrade required validation of another provider missing config map",
			genericProviders: []client.Object{core, provider},
			provider:         awsProvider,
			wantErr:          wrapPhaseError(fmt.Errorf("config map not found"), "config map repository required for validation does not exist yet for provider ns1/aws", operatorv1.ProviderUpgradedCondition).Error(),
		},
		{
			name:             "updgrade requested an unknown provider for the operator",
			genericProviders: []client.Object{core},
			provider:         awsProvider,
			wantErr:          wrapPhaseError(fmt.Errorf("unable to find provider manifest with name aws"), "unable to find generic provider for configclient InfrastructureProvider: aws", operatorv1.ProviderUpgradedCondition).Error(),
		},
		{
			name:               "upgrade required validation of another provider metadata succeeds",
			genericProviders:   []client.Object{core, provider},
			provider:           awsProvider,
			wantDefaultVersion: "v2.3.5",
			wantMetadataSeries: awsMetaReleaseSeries,
			configMaps: []corev1.ConfigMap{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "v2.3.5",
						Namespace: provider.Namespace,
						Labels:    map[string]string{"provider-components": "aws"},
					},
					Data: map[string]string{"metadata": awsMetadata, "components": ""},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			fakeclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(tt.genericProviders...).Build()
			p := &PhaseReconciler{
				ctrlClient:     fakeclient,
				providerConfig: coreProvider,
				repo:           repository.NewMemoryRepository(),
				provider:       core,
			}

			for i := range tt.configMaps {
				g.Expect(fakeclient.Create(ctx, &tt.configMaps[i])).To(Succeed())
			}

			if tt.defaultRepository {
				var err error
				p.repo, err = p.configmapRepository(ctx, &metav1.LabelSelector{
					MatchLabels: map[string]string{
						operatorManagedLabel: "true",
					},
				}, InNamespace("default"))
				g.Expect(err).To(Succeed())
			}

			cl, err := configclient.New(ctx, "")
			g.Expect(err).To(Succeed())

			got, err := p.repositoryProxy(ctx, tt.provider, cl)
			if len(tt.wantErr) > 0 {
				g.Expect(err).Should(MatchError(tt.wantErr))
				return
			}

			g.Expect(err).To(Succeed())

			meta := got.Metadata(tt.wantDefaultVersion)
			metadataData, err := meta.Get(ctx)

			if len(tt.metadataErr) > 0 {
				g.Expect(err).Should(MatchError(tt.metadataErr))
				return
			}

			g.Expect(err).To(Succeed())
			g.Expect(metadataData.ReleaseSeries).To(Equal(tt.wantMetadataSeries))

			g.Expect(got.DefaultVersion()).To(Equal(tt.wantDefaultVersion))
		})
	}
}

func TestRepositoryFactory(t *testing.T) {
	testCases := []struct {
		name          string
		fetchURL      string
		expectedError bool
	}{
		{
			name:     "github repo",
			fetchURL: "https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/v1.4.1/infrastructure-components.yaml",
		},
		{
			name:     "gitlab repo",
			fetchURL: "https://gitlab.example.org/api/v4/projects/group%2Fproject/packages/generic/cluster-api-proviver-aws/v1.4.1/path",
		},
		{
			name:     "gitlab hyphenated repo",
			fetchURL: "https://gitlab-test.example.org/api/v4/projects/group%2Fproject/packages/generic/cluster-api-proviver-aws/v1.4.1/path",
		},
		{
			name:          "unsupported url",
			fetchURL:      "https://unsupported.xyz/kubernetes-sigs/cluster-api-provider-aws/releases/v1.4.1/infrastructure-components.yaml",
			expectedError: true,
		},
		{
			name:          "unsupported schema",
			fetchURL:      "ftp://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/v1.4.1/infrastructure-components.yaml",
			expectedError: true,
		},
		{
			name:          "not an url",
			fetchURL:      "INVALID_URL",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			mr := configclient.NewMemoryReader()

			g.Expect(mr.Init(ctx, "")).To(Succeed())

			var configClient configclient.Client

			var err error

			providerName := "aws"
			providerType := clusterctlv1.InfrastructureProviderType

			// Initialize a client for interacting with the clusterctl configuration.
			// Inject a provider with custom URL.
			if tc.fetchURL != "" {
				reader, err := mr.AddProvider(providerName, providerType, tc.fetchURL)
				g.Expect(err).ToNot(HaveOccurred())

				configClient, err = configclient.New(ctx, "", configclient.InjectReader(reader))
				g.Expect(err).ToNot(HaveOccurred())
			} else {
				configClient, err = configclient.New(ctx, "")
				g.Expect(err).ToNot(HaveOccurred())
			}

			// Get returns the configuration for the provider with a given name/type.
			// This is done using clusterctl internal API types.
			providerConfig, err := configClient.Providers().Get(providerName, providerType)
			g.Expect(err).ToNot(HaveOccurred())

			repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
			if tc.expectedError {
				g.Expect(err).To(HaveOccurred())

				return
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}

			g.Expect(repo.GetVersions(ctx)).To(ContainElement("v1.4.1"))
		})
	}
}

func TestGetLatestVersion(t *testing.T) {
	testCases := []struct {
		name        string
		versions    []string
		expected    string
		expectError bool
	}{
		{
			name:        "Test empty input",
			versions:    []string{},
			expected:    "",
			expectError: true,
		},
		{
			name:        "Test single version",
			versions:    []string{"v1.0.0"},
			expected:    "v1.0.0",
			expectError: false,
		},
		{
			name:        "Test multiple versions",
			versions:    []string{"v1.0.0", "v2.0.0", "v1.5.0"},
			expected:    "v2.0.0",
			expectError: false,
		},
		{
			name:        "Test incorrect versions",
			versions:    []string{"v1.0.0", "NOT_A_VERSION", "v1.5.0"},
			expected:    "",
			expectError: true,
		},
	}

	g := NewWithT(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getLatestVersion(tc.versions)

			if tc.expectError {
				g.Expect(err).To(HaveOccurred())

				return
			}

			g.Expect(got).To(Equal(tc.expected))
			g.Expect(err).ToNot(HaveOccurred())
		})
	}
}
