/*
Copyright 2021 The Kubernetes Authors.

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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

const (
	testMetadata = `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 0
    minor: 4
    contract: v1alpha4
`
	testComponents = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cluster.x-k8s.io/provider: infrastructure-docker
    control-plane: controller-manager
  name: capd-controller-manager
  namespace: capd-system
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/provider: infrastructure-docker
      control-plane: controller-manager
  template:
    metadata:
      labels:
        cluster.x-k8s.io/provider: infrastructure-docker
        control-plane: controller-manager
    spec:
      containers:
      - image: gcr.io/google-samples/hello-app:1.0
        name: manager
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 200m
`
)

func insertDummyConfig(provider operatorv1.GenericProvider) {
	spec := provider.GetSpec()
	spec.FetchConfig = &operatorv1.FetchConfiguration{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test": "dummy-config",
			},
		},
	}
	provider.SetSpec(spec)
}

func dummyConfigMap(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v0.4.2",
			Namespace: ns,
			Labels: map[string]string{
				"test": "dummy-config",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
}

func TestReconcilerPreflightConditions(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		providers []operatorv1.GenericProvider
	}{
		{
			name:      "preflight conditions for CoreProvider",
			namespace: "test-core-provider",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-api",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v0.4.2",
						},
					},
				},
			},
		},
		{
			name:      "preflight conditions for ControlPlaneProvider",
			namespace: "test-cp-provider",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-api",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v0.4.2",
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []clusterv1.Condition{
								{
									Type:   clusterv1.ReadyCondition,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
				&operatorv1.ControlPlaneProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubeadm",
					},
					Spec: operatorv1.ControlPlaneProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v0.4.2",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			t.Log("creating namespace", tc.namespace)
			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tc.namespace}}
			g.Expect(env.CreateAndWait(ctx, namespace)).To(Succeed())
			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(tc.namespace))).To(Succeed())

			for _, p := range tc.providers {
				insertDummyConfig(p)
				p.SetNamespace(tc.namespace)
				t.Log("creating test provider", p.GetName())
				g.Expect(env.CreateAndWait(ctx, p)).To(Succeed())
			}

			g.Eventually(func() bool {
				for _, p := range tc.providers {
					if err := env.Get(ctx, client.ObjectKeyFromObject(p), p); err != nil {
						return false
					}

					for _, cond := range p.GetStatus().Conditions {
						if cond.Type == operatorv1.PreflightCheckCondition {
							t.Log(t.Name(), p.GetName(), cond)
							if cond.Status == corev1.ConditionTrue {
								return true
							}
						}
					}
				}

				return false
			}, timeout).Should(BeEquivalentTo(true))

			objs := []client.Object{}
			for _, p := range tc.providers {
				objs = append(objs, p)
			}
			g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
		})
	}
}

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme))
	return scheme
}
