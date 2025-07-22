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

package controller

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfigMapChangesAreAppliedToTheProvider(t *testing.T) {
	g := NewWithT(t)
	objs := []client.Object{}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
	}()

	ns, err := env.CreateNamespace(ctx, "configmap-changes-namespace")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	t.Log("Ensure namespace exists", ns.Name)

	// Create ConfigMap with initial content
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCurrentVersion,
			Namespace: ns.Name,
			Labels: map[string]string{
				"provider-components": "edge",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
	g.Expect(env.CreateAndWait(ctx, configMap)).To(Succeed())
	objs = append(objs, configMap)

	t.Log("Created ConfigMap")

	// Create CoreProvider first (required for InfrastructureProvider)
	coreProvider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, coreProvider)).To(Succeed())
	objs = append(objs, coreProvider)

	// Wait for CoreProvider to be installed
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider); err != nil {
			return false
		}
		return conditions.IsTrue(coreProvider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeTrue())

	t.Log("CoreProvider is installed")

	// Manually set ReadyCondition as it's not set automatically in test env
	patchHelper, err := patch.NewHelper(coreProvider, env)
	g.Expect(err).ToNot(HaveOccurred())
	conditions.MarkTrue(coreProvider, clusterv1.ReadyCondition)
	g.Expect(patchHelper.Patch(ctx, coreProvider)).To(Succeed())

	// Create InfrastructureProvider that uses the ConfigMap
	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "edge-provider-",
			Namespace:    ns.Name,
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"provider-components": "edge",
						},
					},
				},
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())
	objs = append(objs, provider)

	t.Log("Created provider")

	// Wait for provider to be ready
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		return conditions.IsTrue(provider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeTrue())

	t.Log("Provider is installed")

	// Wait for the provider to have a hash annotation (this happens after full reconciliation)
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}
		hash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		return hash != ""
	}, timeout).Should(BeTrue(), "Provider should have a hash annotation after reconciliation")

	// Get the initial hash annotation
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
	initialHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
	g.Expect(initialHash).ToNot(BeEmpty())

	t.Log("Initial hash:", initialHash)

	// Update the ConfigMap content
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)).To(Succeed())
	configMap.Data["components"] = testComponents + "\n# Updated content"
	g.Expect(env.Update(ctx, configMap)).To(Succeed())

	t.Log("ConfigMap updated")

	// Wait for provider to be reconciled with new hash
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		currentHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		return currentHash != "" && currentHash != initialHash
	}, 30*time.Second).Should(BeTrue())

	t.Log("Provider reconciled with new hash")
}

func TestConfigMapChangesWithNonMatchingSelector(t *testing.T) {
	g := NewWithT(t)
	objs := []client.Object{}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
	}()

	ns, err := env.CreateNamespace(ctx, "non-matching-selector-namespace")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	t.Log("Ensure namespace exists", ns.Name)

	// Create CoreProvider first (required for InfrastructureProvider)
	coreProvider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, coreProvider)).To(Succeed())
	objs = append(objs, coreProvider)

	// Wait for CoreProvider to be installed
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider); err != nil {
			return false
		}
		return conditions.IsTrue(coreProvider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeTrue())

	t.Log("CoreProvider is installed")

	// Manually set ReadyCondition as it's not set automatically in test env
	patchHelper, err := patch.NewHelper(coreProvider, env)
	g.Expect(err).ToNot(HaveOccurred())
	conditions.MarkTrue(coreProvider, clusterv1.ReadyCondition)
	g.Expect(patchHelper.Patch(ctx, coreProvider)).To(Succeed())

	// Create ConfigMap that won't match any provider selector
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "unrelated-configmap-",
			Namespace:    ns.Name,
			Labels: map[string]string{
				"provider-components": "unrelated",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
	g.Expect(env.CreateAndWait(ctx, configMap)).To(Succeed())
	objs = append(objs, configMap)

	// Create provider that uses different selector
	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "provider-",
			Namespace:    ns.Name,
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"provider-components": "specific",
						},
					},
				},
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())
	objs = append(objs, provider)

	// Create ConfigMap that matches the provider selector
	matchingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCurrentVersion,
			Namespace: ns.Name,
			Labels: map[string]string{
				"provider-components": "specific",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
	g.Expect(env.CreateAndWait(ctx, matchingConfigMap)).To(Succeed())
	objs = append(objs, matchingConfigMap)

	t.Log("Created provider with matching and non-matching ConfigMaps")

	// Wait for provider to be ready
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		return conditions.IsTrue(provider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeTrue())

	// Wait for the provider to have a hash annotation (this happens after full reconciliation)
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}
		hash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		return hash != ""
	}, timeout).Should(BeTrue(), "Provider should have a hash annotation after reconciliation")

	// Get initial hash
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
	initialHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
	g.Expect(initialHash).ToNot(BeEmpty())

	// Update the non-matching ConfigMap - this should NOT trigger provider reconciliation
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)).To(Succeed())
	configMap.Data["components"] = testComponents + "\n# Updated unrelated content"
	g.Expect(env.Update(ctx, configMap)).To(Succeed())

	t.Log("Updated non-matching ConfigMap")

	// Wait a bit and verify the provider hash hasn't changed
	g.Consistently(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		currentHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		return currentHash == initialHash
	}, 10*time.Second, 2*time.Second).Should(BeTrue())

	t.Log("Provider hash remained unchanged as expected")

	// Now update the matching ConfigMap - this SHOULD trigger provider reconciliation
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(matchingConfigMap), matchingConfigMap)).To(Succeed())
	matchingConfigMap.Data["components"] = testComponents + "\n# Updated matching content"
	g.Expect(env.Update(ctx, matchingConfigMap)).To(Succeed())

	t.Log("Updated matching ConfigMap")

	// Wait for provider to be reconciled with new hash
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		currentHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		return currentHash != "" && currentHash != initialHash
	}, 30*time.Second).Should(BeTrue())

	t.Log("Provider reconciled with new hash after matching ConfigMap update")
}

func TestMultipleConfigMapsError(t *testing.T) {
	g := NewWithT(t)
	objs := []client.Object{}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
	}()

	ns, err := env.CreateNamespace(ctx, "multiple-configmaps-error-namespace")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	t.Log("Ensure namespace exists", ns.Name)

	// Create CoreProvider first (required for InfrastructureProvider)
	coreProvider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, coreProvider)).To(Succeed())
	objs = append(objs, coreProvider)

	// Wait for CoreProvider to be installed
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider); err != nil {
			return false
		}
		return conditions.IsTrue(coreProvider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeTrue())

	t.Log("CoreProvider is installed")

	// Manually set ReadyCondition as it's not set automatically in test env
	patchHelper, err := patch.NewHelper(coreProvider, env)
	g.Expect(err).ToNot(HaveOccurred())
	conditions.MarkTrue(coreProvider, clusterv1.ReadyCondition)
	g.Expect(patchHelper.Patch(ctx, coreProvider)).To(Succeed())

	// Create multiple ConfigMaps with the same labels (this should cause an error)
	configMap1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCurrentVersion,
			Namespace: ns.Name,
			Labels: map[string]string{
				"provider-components": "edge",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
	g.Expect(env.CreateAndWait(ctx, configMap1)).To(Succeed())
	objs = append(objs, configMap1)

	configMap2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1.1.0",
			Namespace: ns.Name,
			Labels: map[string]string{
				"provider-components": "edge",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents + "\n# Different content",
		},
	}
	g.Expect(env.CreateAndWait(ctx, configMap2)).To(Succeed())
	objs = append(objs, configMap2)

	t.Log("Created multiple ConfigMaps with same labels")

	// Create InfrastructureProvider that uses the ConfigMaps (should fail due to multiple matches)
	provider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "edge-provider-",
			Namespace:    ns.Name,
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"provider-components": "edge",
						},
					},
				},
			},
		},
	}
	g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())
	objs = append(objs, provider)

	t.Log("Created provider with selector matching multiple ConfigMaps")

	// Provider should have error condition due to multiple ConfigMaps
	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}
		return conditions.IsFalse(provider, operatorv1.ProviderInstalledCondition) &&
			conditions.GetReason(provider, operatorv1.ProviderInstalledCondition) != ""
	}, timeout).Should(BeTrue())

	t.Log("Provider correctly failed with multiple ConfigMaps error")
}
