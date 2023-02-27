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

package controllers

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api-operator/internal/controllers/genericprovider"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretReader(t *testing.T) {
	g := NewWithT(t)

	fakeclient := fake.NewClientBuilder().WithObjects().Build()

	secretName := "test-secret"
	namespace := "test-namespace"

	p := &phaseReconciler{
		ctrlClient: fakeclient,
		provider: &genericprovider.CoreProviderWrapper{
			CoreProvider: &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-api",
					Namespace: namespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "CoreProvider",
					APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						SecretName: secretName,
						FetchConfig: &operatorv1.FetchConfiguration{
							URL: "https://example.com",
						},
					},
				},
			},
		},
	}

	testKey1 := "test-key1"
	testValue1 := "test-value1"
	testKey2 := "test-key2"
	testValue2 := "test-value2"

	g.Expect(fakeclient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			testKey1: []byte(testValue1),
			testKey2: []byte(testValue2),
		},
	})).To(Succeed())

	configreader, err := p.secretReader(context.TODO())
	g.Expect(err).ToNot(HaveOccurred())

	expectedValue1, err := configreader.Get(testKey1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(expectedValue1).To(Equal(testValue1))

	expectedValue2, err := configreader.Get(testKey2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(expectedValue2).To(Equal(testValue2))

	exptectedProviderData, err := configreader.Get("providers")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exptectedProviderData).To(Equal("- name: cluster-api\n  type: CoreProvider\n  url: https://example.com\n"))
}

func TestConfigmapRepository(t *testing.T) {
	provider := &genericprovider.InfrastructureProviderWrapper{
		InfrastructureProvider: &operatorv1.InfrastructureProvider{
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
	tests := []struct {
		name               string
		configMaps         []corev1.ConfigMap
		want               repository.Repository
		wantErr            string
		wantDefaultVersion string
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
			wantErr: "ConfigMap ns1/v1.2.3 has no components",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			fakeclient := fake.NewClientBuilder().WithScheme(setupScheme()).WithObjects(provider.GetObject()).Build()
			p := &phaseReconciler{
				ctrlClient: fakeclient,
				provider:   provider,
			}

			for i := range tt.configMaps {
				g.Expect(fakeclient.Create(ctx, &tt.configMaps[i])).To(Succeed())
			}

			got, err := p.configmapRepository(context.TODO())
			if len(tt.wantErr) > 0 {
				g.Expect(err).Should(MatchError(tt.wantErr))
				return
			}
			g.Expect(err).To(Succeed())
			g.Expect(got.GetFile(got.DefaultVersion(), got.ComponentsPath())).To(Equal([]byte(components)))
			g.Expect(got.GetFile(got.DefaultVersion(), "metadata.yaml")).To(Equal([]byte(metadata)))
			g.Expect(got.DefaultVersion()).To(Equal(tt.wantDefaultVersion))
		})
	}
}
