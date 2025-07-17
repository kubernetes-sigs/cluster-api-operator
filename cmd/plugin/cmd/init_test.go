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

package cmd

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
)

func TestCheckCAPIOperatorAvailability(t *testing.T) {
	tests := []struct {
		name    string
		want    bool
		wantErr bool
	}{
		{
			name:    "no deployment",
			want:    false,
			wantErr: false,
		},
		{
			name:    "with deployment",
			want:    true,
			wantErr: false,
		},
		{
			name:    "two deployments",
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			resources := []ctrlclient.Object{}

			if tt.want {
				deployment := generateCAPIOperatorDeployment("capi-operator-controller-manager", "default")

				g.Expect(env.Create(ctx, deployment)).To(Succeed())

				// Get created deployment and update its status
				g.Eventually(func() (bool, error) {
					err := env.Get(ctx, ctrlclient.ObjectKeyFromObject(deployment), deployment)
					if err != nil {
						return false, err
					}

					return deployment != nil, nil
				}, waitShort).Should(BeTrue())

				deployment.Status.Conditions = []appsv1.DeploymentCondition{
					{
						Type:    appsv1.DeploymentAvailable,
						Status:  corev1.ConditionTrue,
						Reason:  "MinimumReplicasAvailable",
						Message: "Deployment has minimum availability.",
					},
				}

				g.Expect(env.Status().Update(ctx, deployment)).To(Succeed())

				g.Eventually(func() (bool, error) {
					deploymentFromServer := &appsv1.Deployment{}
					err := env.Get(ctx, ctrlclient.ObjectKeyFromObject(deployment), deploymentFromServer)
					if err != nil {
						return false, err
					}

					return deploymentFromServer != nil, nil
				}, waitShort).Should(BeTrue())

				resources = append(resources, deployment)
			}

			if tt.wantErr {
				// To generate an error we create two deployments with the same labels.
				// Deployment 1.
				deployment := generateCAPIOperatorDeployment("capi-operator-controller-manager", "default")
				resources = append(resources, deployment)

				g.Expect(env.Create(ctx, deployment)).To(Succeed())

				g.Eventually(func() (bool, error) {
					deploymentFromServer := &appsv1.Deployment{}
					err := env.Get(ctx, ctrlclient.ObjectKeyFromObject(deployment), deploymentFromServer)
					if err != nil {
						return false, err
					}

					return deploymentFromServer != nil, nil
				}, waitShort).Should(BeTrue())

				resources = append(resources, deployment)

				// Deployment 2.
				deploymentClone := generateCAPIOperatorDeployment("capi-operator-controller-manager-clone", "default")

				g.Expect(env.Create(ctx, deploymentClone)).To(Succeed())

				g.Eventually(func() (bool, error) {
					deploymentFromServer := &appsv1.Deployment{}
					err := env.Get(ctx, ctrlclient.ObjectKeyFromObject(deploymentClone), deploymentFromServer)
					if err != nil {
						return false, err
					}

					return deploymentFromServer != nil, nil
				}, waitShort).Should(BeTrue())

				resources = append(resources, deploymentClone)
			}

			available, err := CheckDeploymentAvailability(ctx, env, capiOperatorLabels)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(available).To(Equal(tt.want))
			}

			g.Expect(env.CleanupAndWait(ctx, resources...)).To(Succeed())
		})
	}
}

func TestInitProviders(t *testing.T) {
	tests := []struct {
		name            string
		opts            *initOptions
		wantedProviders []genericprovider.GenericProvider
		wantErr         bool
	}{
		{
			name:            "no providers",
			wantedProviders: []genericprovider.GenericProvider{},
			wantErr:         false,
			opts:            &initOptions{},
		},
		{
			name: "core provider",
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
			name: "core provider in default target namespace",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-operator-system", "v1.8.0", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api::v1.8.0",
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name: "core provider without version",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-system", "", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api:capi-system",
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name: "core provider without namespace and version",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-operator-system", "", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api",
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name: "core provider with config secret",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-operator-system", "", "capi-secrets", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api",
				targetNamespace: "capi-operator-system",
				configSecret:    "capi-secrets",
			},
		},
		{
			name: "core provider with config secret in a custom namespace",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-operator-system", "", "capi-secrets", "custom-namespace"),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider:    "cluster-api",
				targetNamespace: "capi-operator-system",
				configSecret:    "capi-secrets:custom-namespace",
			},
		},
		{
			name: "multiple providers of one type",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "aws", "capa-operator-system", "", "", ""),
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "capd-operator-system", "", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				infrastructureProviders: []string{
					"aws:capa-operator-system",
					"docker:capd-operator-system",
				},
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name: "all providers",
			wantedProviders: []genericprovider.GenericProvider{
				generateGenericProvider(clusterctlv1.CoreProviderType, "cluster-api", "capi-system", "v1.8.0", "", ""),
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "aws", "capa-operator-system", "", "", ""),
				generateGenericProvider(clusterctlv1.InfrastructureProviderType, "docker", "capd-operator-system", "", "", ""),
				generateGenericProvider(clusterctlv1.ControlPlaneProviderType, "kubeadm", "kcp-system", "", "", ""),
				generateGenericProvider(clusterctlv1.BootstrapProviderType, "kubeadm", "bootstrap-system", "", "", ""),
				generateGenericProvider(clusterctlv1.AddonProviderType, "helm", "caaph-system", "", "", ""),
			},
			wantErr: false,
			opts: &initOptions{
				coreProvider: "cluster-api:capi-system:v1.8.0",
				infrastructureProviders: []string{
					"aws:capa-operator-system",
					"docker:capd-operator-system",
				},
				controlPlaneProviders: []string{
					"kubeadm:kcp-system",
				},
				bootstrapProviders: []string{
					"kubeadm:bootstrap-system",
				},
				addonProviders: []string{
					"helm:caaph-system",
				},
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name:            "invalid input",
			wantedProviders: []genericprovider.GenericProvider{},
			wantErr:         true,
			opts: &initOptions{
				infrastructureProviders: []string{
					"TOO:MANY:PARTS:HERE",
				},
				targetNamespace: "capi-operator-system",
			},
		},
		{
			name:            "empty provider",
			wantedProviders: []genericprovider.GenericProvider{},
			wantErr:         true,
			opts: &initOptions{
				infrastructureProviders: []string{
					"",
				},
				targetNamespace: "capi-operator-system",
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

			g.Expect(env.CleanupAndWait(ctx, resources...)).To(Succeed())
		})
	}
}

func generateCAPIOperatorDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    capiOperatorLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: capiOperatorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: capiOperatorLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.14.2",
						},
					},
				},
			},
		},
	}
}

func generateGenericProvider(providerType clusterctlv1.ProviderType, name, namespace, version, configSecretName, configSecretNamespace string) genericprovider.GenericProvider {
	genericProvider := NewGenericProvider(providerType)

	genericProvider.SetName(name)

	genericProvider.SetNamespace(namespace)

	spec := genericProvider.GetSpec()
	spec.Version = version
	spec.ConfigSecret = &operatorv1.SecretReference{
		Name:      configSecretName,
		Namespace: configSecretNamespace,
	}
	genericProvider.SetSpec(spec)

	return genericProvider
}

func getGenericProvider(ctx context.Context, client ctrlclient.Client, providerKind, providerName, providerNamespace string) (genericprovider.GenericProvider, error) {
	switch providerKind {
	case "CoreProvider":
		provider := &operatorv1.CoreProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	case "BootstrapProvider":
		provider := &operatorv1.BootstrapProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	case "ControlPlaneProvider":
		provider := &operatorv1.ControlPlaneProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	case "InfrastructureProvider":
		provider := &operatorv1.InfrastructureProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	case "AddonProvider":
		provider := &operatorv1.AddonProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	case "RuntimeExtensionProvider":
		provider := &operatorv1.RuntimeExtensionProvider{}
		if err := client.Get(ctx, types.NamespacedName{Name: providerName, Namespace: providerNamespace}, provider); err != nil {
			return nil, err
		}

		return provider, nil
	default:
		return nil, fmt.Errorf("failed to cast interface for type: %s", providerKind)
	}
}
