/*
Copyright 2024 The Kubernetes Authors.

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
	"testing"

	. "github.com/onsi/gomega"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
)

func TestUpgradePlan(t *testing.T) {
	tests := []struct {
		name              string
		opts              *initOptions
		customURL         string
		wantedUpgradePlan upgradePlan
		wantedProviders   []genericprovider.GenericProvider
		wantErr           bool
	}{
		{
			name: "no providers",
			wantedUpgradePlan: upgradePlan{
				Contract:  "v1beta1",
				Providers: []upgradeItem{},
			},
			wantErr: false,
			opts:    &initOptions{},
		},
		{
			name: "builtin core provider",
			wantedUpgradePlan: upgradePlan{
				Contract: "v1beta1",
				Providers: []upgradeItem{
					{
						Name:           "cluster-api",
						Namespace:      "capi-system",
						Type:           "core",
						CurrentVersion: "v1.8.0",
						Source:         "https://github.com/kubernetes-sigs/cluster-api/releases/latest/core-components.yaml",
						SourceType:     providerSourceTypeBuiltin,
					},
				},
			},
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-system", "v1.8.0", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api:capi-system:v1.8.0",
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name:      "custom infra provider",
			customURL: "https://github.com/kubernetes-sigs/cluster-api/releases/latest/core-components.yaml",
			wantedUpgradePlan: upgradePlan{
				Contract: "v1beta1",
				Providers: []upgradeItem{
					{
						Name:           "docker",
						Namespace:      "capi-system",
						Type:           "infrastructure",
						CurrentVersion: "v1.8.0",
						Source:         "https://github.com/kubernetes-sigs/cluster-api/releases/latest/core-components.yaml",
						SourceType:     providerSourceTypeCustomURL,
					},
				},
			},
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "capi-system", "v1.8.0", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				infrastructureProviders: []string{"docker:capi-system:v1.8.0"},
				targetNamespace:         "capi-operator-system",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			resources := []ctrlclient.Object{}

			for _, provider := range tt.wantedProviders {
				resources = append(resources, provider)
			}

			err := initProviders(ctx, env, tt.opts)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())

				return
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}

			for _, genericProvider := range tt.wantedProviders {
				g.Eventually(func() (bool, error) {
					provider, err := getGenericProvider(ctx, env, string(util.ClusterctlProviderType(genericProvider)), genericProvider.ProviderName(), genericProvider.GetNamespace())
					if err != nil {
						return false, err
					}

					if provider.GetSpec().Version != genericProvider.GetSpec().Version {
						return false, nil
					}

					return true, nil
				}, waitShort).Should(BeTrue())
			}

			// Init doesn't support custom URLs yet, so we have to update providers here
			if tt.customURL != "" {
				for _, genericProvider := range tt.wantedProviders {
					provider, err := getGenericProvider(ctx, env, string(util.ClusterctlProviderType(genericProvider)), genericProvider.ProviderName(), genericProvider.GetNamespace())
					g.Expect(err).NotTo(HaveOccurred())

					spec := provider.GetSpec()
					spec.FetchConfig = &operatorv1.FetchConfiguration{
						URL: tt.customURL,
					}

					provider.SetSpec(spec)

					g.Expect(env.Update(ctx, provider)).NotTo(HaveOccurred())

					g.Eventually(func() (bool, error) {
						provider, err := getGenericProvider(ctx, env, string(util.ClusterctlProviderType(genericProvider)), genericProvider.ProviderName(), genericProvider.GetNamespace())
						if err != nil {
							return false, err
						}

						if provider.GetSpec().FetchConfig == nil || provider.GetSpec().FetchConfig.URL != tt.customURL {
							return false, nil
						}

						return true, nil
					}, waitShort).Should(BeTrue())
				}
			}

			// Run upgrade plan
			upgradePlan, err := planUpgrade(ctx, env)
			g.Expect(err).NotTo(HaveOccurred())

			for i, provider := range upgradePlan.Providers {
				g.Expect(provider.Name).To(Equal(tt.wantedUpgradePlan.Providers[i].Name))
				g.Expect(provider.Namespace).To(Equal(tt.wantedUpgradePlan.Providers[i].Namespace))
				g.Expect(provider.Type).To(Equal(tt.wantedUpgradePlan.Providers[i].Type))
				g.Expect(provider.CurrentVersion).To(Equal(tt.wantedUpgradePlan.Providers[i].CurrentVersion))
				g.Expect(provider.Source).To(Equal(tt.wantedUpgradePlan.Providers[i].Source))
				g.Expect(provider.SourceType).To(Equal(tt.wantedUpgradePlan.Providers[i].SourceType))
			}

			g.Expect(env.CleanupAndWait(ctx, resources...)).To(Succeed())
		})
	}
}
