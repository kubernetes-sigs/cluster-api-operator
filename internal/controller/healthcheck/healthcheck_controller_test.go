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

package healthcheck

import (
	"testing"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

const (
	testMetadata = `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 1
    minor: 14
    contract: v1beta2
`
	testComponents = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cluster.x-k8s.io/provider: cluster-api
    control-plane: controller-manager
  name: capi-controller-manager
  namespace: capi-system
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/provider: cluster-api
      control-plane: controller-manager
  template:
    metadata:
      labels:
        cluster.x-k8s.io/provider: cluster-api
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

	testCurrentVersion = "v1.14.0"
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

func dummyConfigMap(ns, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
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

func TestReconcilerReadyConditionOnProviderChange(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "ready-provider-change")
	g.Expect(err).ToNot(HaveOccurred())

	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker",
			Namespace: ns.Name,
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())

	conditions.Set(provider, metav1.Condition{
		Type:   operatorv1.ProviderInstalledCondition,
		Status: metav1.ConditionTrue,
		Reason: "ProviderInstalled",
	})
	g.Expect(env.Status().Update(ctx, provider)).To(Succeed())
	g.Eventually(func(g Gomega) {
		g.Expect(env.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		g.Expect(conditions.IsTrue(provider, operatorv1.ProviderInstalledCondition)).To(BeTrue())
	}, timeout).Should(Succeed())

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "capd-controller-manager",
			Namespace: ns.Name,
			Labels:    map[string]string{providerLabelKey: "infrastructure-docker"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: operatorv1.GroupVersion.String(),
				Kind:       "InfrastructureProvider",
				Name:       provider.Name,
				UID:        provider.UID,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "capd"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "capd"}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "manager", Image: "capd:test"}}},
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, deployment)).To(Succeed())

	deployment.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue}}
	g.Expect(env.Status().Update(ctx, deployment)).To(Succeed())

	g.Eventually(func(g Gomega) {
		g.Expect(env.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		g.Expect(conditions.IsTrue(provider, clusterv1.ReadyCondition)).To(BeTrue())
	}, timeout).Should(Succeed())

	// Only the provider changes here, so only the provider watch can trigger the health check.
	patch := client.MergeFrom(provider.DeepCopy())
	conditions.Delete(provider, clusterv1.ReadyCondition)
	g.Expect(env.Status().Patch(ctx, provider, patch)).To(Succeed())

	// Read from the API server, as the cache may still hold the Ready condition from before the patch.
	g.Eventually(func(g Gomega) {
		g.Expect(env.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		g.Expect(conditions.IsTrue(provider, clusterv1.ReadyCondition)).To(BeTrue())
	}, timeout).Should(Succeed())

	g.Expect(env.CleanupAndWait(ctx, deployment, provider)).To(Succeed())
}

func TestReconcilerReadyConditions(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedAvailability corev1.ConditionStatus
	}{
		{
			name:                 "correct CoreProvider",
			expectedAvailability: corev1.ConditionTrue,
		},
		{
			name:                 "invalid CoreProvider",
			expectedAvailability: corev1.ConditionFalse,
		},
	}

	namespace := "capi-system"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-api",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: testCurrentVersion,
					},
				},
			}

			g.Expect(env.EnsureNamespaceExists(ctx, namespace)).To(Succeed())

			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(namespace, testCurrentVersion))).To(Succeed())

			insertDummyConfig(provider)
			provider.SetNamespace(namespace)

			g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())

			g.Eventually(func() bool {
				deployment := &appsv1.Deployment{}

				if err := env.Client.Get(ctx, types.NamespacedName{
					Name:      "capi-controller-manager",
					Namespace: namespace,
				}, deployment); err != nil {
					return false
				}

				deployment.Status.Conditions = []appsv1.DeploymentCondition{
					{
						Type:   appsv1.DeploymentAvailable,
						Status: tc.expectedAvailability,
					},
				}

				if err := env.Status().Update(ctx, deployment); err != nil {
					return false
				}

				return true
			}, timeout).Should(BeTrue())

			g.Eventually(func() bool {
				if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
					return false
				}

				for _, cond := range provider.GetStatus().Conditions {
					if cond.Type == clusterv1.ReadyCondition {
						t.Log(t.Name(), provider.GetName(), cond)

						if cond.Status == metav1.ConditionStatus(tc.expectedAvailability) {
							return true
						}
					}
				}

				return false
			}, timeout).Should(BeTrue())

			objs := make([]client.Object, 0, 2)
			objs = append(objs, provider)

			objs = append(objs, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testCurrentVersion,
					Namespace: namespace,
				},
			})

			g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
		})
	}
}
