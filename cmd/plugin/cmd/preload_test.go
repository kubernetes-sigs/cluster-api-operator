/*
Copyright 2025 The Kubernetes Authors.

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

package cmd

import (
	"cmp"
	"os"
	"path"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
)

type publishProvider struct {
	configMapName  string
	provider       genericprovider.GenericProvider
	metadataKey    string
	componentsKey  string
	metadataData   []byte
	componentsData []byte
}

type publishOptions struct {
	ociUrl    string
	providers []publishProvider
}

func TestPreloadCommand(t *testing.T) {
	tests := []struct {
		name               string
		customURL          string
		publishOpts        *publishOptions
		existingProviders  []genericprovider.GenericProvider
		expectedConfigMaps int
		wantErr            bool
	}{
		{
			name:    "no providers",
			wantErr: false,
		},
		{
			name: "builtin core provider with OCI override",
			publishOpts: &publishOptions{
				ociUrl: "ttl.sh/cluster-api-operator-manifests:1m",
				providers: []publishProvider{{
					configMapName:  "core-cluster-api-v1.9.3",
					provider:       generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "default", "v1.9.3", "", ""),
					metadataKey:    "metadata.yaml",
					metadataData:   []byte("metadata"),
					componentsKey:  "components.yaml",
					componentsData: []byte("components"),
				}},
			},
			expectedConfigMaps: 1,
		},
		{
			name: "multiple providers with OCI override",
			publishOpts: &publishOptions{
				ociUrl: "ttl.sh/cluster-api-operator-manifests:1m",
				providers: []publishProvider{{
					configMapName:  "core-cluster-api-v1.9.3",
					provider:       generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "default", "v1.9.3", "", ""),
					metadataKey:    "core-cluster-api-v1.9.3-metadata.yaml",
					metadataData:   []byte("metadata"),
					componentsKey:  "core-cluster-api-v1.9.3-components.yaml",
					componentsData: []byte("components"),
				}, {
					configMapName:  "infrastructure-docker-v1.9.3",
					provider:       generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "default", "v1.9.3", "", ""),
					metadataKey:    "infrastructure-docker-v1.9.3-metadata.yaml",
					metadataData:   []byte("metadata"),
					componentsKey:  "infrastructure-docker-v1.9.3-components.yaml",
					componentsData: []byte("components"),
				}},
			},
			expectedConfigMaps: 2,
		},
		{
			name: "custom url infra provider",
			existingProviders: []genericprovider.GenericProvider{
				func() genericprovider.GenericProvider {
					p := generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "default", "v1.9.3", "", "")
					spec := p.GetSpec()
					spec.FetchConfig = &operatorv1.FetchConfiguration{
						URL: "https://github.com/kubernetes-sigs/cluster-api/releases/latest/core-components.yaml",
					}
					p.SetSpec(spec)

					return p
				}(),
			},
			expectedConfigMaps: 1,
		},
		{
			name: "regular core and infra provider",
			existingProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "default", "v1.9.3", "", ""),
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "default", "v1.9.3", "", ""),
			},
			expectedConfigMaps: 2,
		},
		{
			name: "OCI override with incorrect metadata key",
			publishOpts: &publishOptions{
				ociUrl: "ttl.sh/cluster-api-operator-manifests:1m",
				providers: []publishProvider{{
					configMapName:  "core-cluster-api-v1.9.3",
					provider:       generateGenericProvider(clusterctlv1.InfrastructureProviderType, "metadata-missing", "default", "v1.9.3", "", ""),
					metadataKey:    "incorrect-metadata.yaml",
					metadataData:   []byte("test"),
					componentsKey:  "components.yaml",
					componentsData: []byte("test"),
				}},
			},
			wantErr: true,
		},
		{
			name: "OCI override with incorrect components key",
			publishOpts: &publishOptions{
				ociUrl: "ttl.sh/cluster-api-operator-manifests:1m",
				providers: []publishProvider{{
					configMapName:  "core-cluster-api-v1.9.3",
					provider:       generateGenericProvider(clusterctlv1.InfrastructureProviderType, "components-missing", "default", "v1.9.3", "", ""),
					metadataKey:    "metadata.yaml",
					metadataData:   []byte("test"),
					componentsKey:  "incorrect-components.yaml",
					componentsData: []byte("test"),
				}},
			},
			wantErr: true,
		},
		{
			name: "OCI override with missing image",
			existingProviders: []genericprovider.GenericProvider{
				func() genericprovider.GenericProvider {
					p := generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "default", "v1.9.3", "", "")
					spec := p.GetSpec()
					spec.FetchConfig = &operatorv1.FetchConfiguration{
						OCIConfiguration: operatorv1.OCIConfiguration{
							OCI: "ttl.sh/cluster-api-operator-manifests-missing:1m",
						},
					}
					p.SetSpec(spec)

					return p
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			dir, err := os.MkdirTemp("", "manifests")
			defer func() {
				g.Expect(os.RemoveAll(dir)).To(Succeed())
			}()
			g.Expect(err).To(Succeed())

			opts := cmp.Or(tt.publishOpts, &publishOptions{})
			if tt.publishOpts != nil && opts.ociUrl != "" {
				for _, provider := range opts.providers {
					err = os.WriteFile(path.Join(dir, provider.metadataKey), provider.metadataData, 0o777)
					g.Expect(err).To(Succeed())
					err = os.WriteFile(path.Join(dir, provider.componentsKey), provider.componentsData, 0o777)
					g.Expect(err).To(Succeed())
				}

				g.Expect(publish(ctx, dir, opts.ociUrl)).To(Succeed())

				for _, data := range opts.providers {
					spec := data.provider.GetSpec()
					spec.FetchConfig = &operatorv1.FetchConfiguration{
						OCIConfiguration: operatorv1.OCIConfiguration{
							OCI: opts.ociUrl,
						},
					}
					data.provider.SetSpec(spec)
					g.Expect(env.Client.Create(ctx, data.provider)).To(Succeed())
				}
			}

			resources := []ctrlclient.Object{}
			for _, provider := range tt.existingProviders {
				resources = append(resources, provider)
			}

			for _, data := range opts.providers {
				resources = append(resources, data.provider)
			}

			defer func() {
				g.Expect(env.CleanupAndWait(ctx, resources...)).To(Succeed())
			}()

			for _, genericProvider := range tt.existingProviders {
				g.Expect(env.Client.Create(ctx, genericProvider)).To(Succeed())
			}

			configMaps := []*corev1.ConfigMap{}

			g.Eventually(func(g Gomega) {
				configMaps, err = preloadExisting(ctx, env)
				g.Expect(tt.expectedConfigMaps).To(Equal(len(configMaps)))
				if tt.wantErr {
					g.Expect(err).To(HaveOccurred())
				} else {
					g.Expect(err).NotTo(HaveOccurred())

					maps := map[types.NamespacedName]*corev1.ConfigMap{}
					for _, cm := range configMaps {
						maps[ctrlclient.ObjectKeyFromObject(cm)] = cm
					}

					for _, data := range opts.providers {
						cm, ok := maps[types.NamespacedName{
							Namespace: data.provider.GetNamespace(),
							Name:      data.configMapName,
						}]

						g.Expect(ok).To(BeTrue())

						g.Expect(cm.Data["metadata"]).To(Equal(string(data.metadataData)))
						g.Expect(cm.Data["components"]).To(Equal(string(data.componentsData)))
					}
				}
			}, "15s", "1s").Should(Succeed())
		})
	}
}
