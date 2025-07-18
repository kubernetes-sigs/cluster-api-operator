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
	"sigs.k8s.io/cluster-api/util/conditions"
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
			GenerateName: "edge-provider-",
			Namespace:    ns.Name,
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

func TestConfigMapChangesWithMultipleProviders(t *testing.T) {
	g := NewWithT(t)
	objs := []client.Object{}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
	}()

	ns, err := env.CreateNamespace(ctx, "multiple-providers-namespace")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	t.Log("Ensure namespace exists", ns.Name)

	// Create ConfigMap with initial content
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "shared-configmap-",
			Namespace:    ns.Name,
			Labels: map[string]string{
				"provider-components": "shared",
				"environment":         "test",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
	g.Expect(env.CreateAndWait(ctx, configMap)).To(Succeed())
	objs = append(objs, configMap)

	t.Log("Created shared ConfigMap")

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

	// Create multiple providers that use the same ConfigMap
	providers := []*operatorv1.InfrastructureProvider{}
	for i := 0; i < 2; i++ {
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
								"provider-components": "shared",
							},
						},
					},
				},
			},
		}
		g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())
		objs = append(objs, provider)
		providers = append(providers, provider)
	}

	t.Log("Created multiple providers")

	// Wait for all providers to be ready
	for _, provider := range providers {
		g.Eventually(func() bool {
			if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
				return false
			}

			return conditions.IsTrue(provider, operatorv1.ProviderInstalledCondition)
		}, timeout).Should(BeTrue())
	}

	t.Log("All providers are installed")

	// Get initial hash annotations
	initialHashes := make(map[string]string)
	for i, provider := range providers {
		g.Expect(env.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		hash := provider.GetAnnotations()[appliedSpecHashAnnotation]
		g.Expect(hash).ToNot(BeEmpty())
		initialHashes[provider.GetName()] = hash
		t.Logf("Provider %d initial hash: %s", i, hash)
	}

	// Update the ConfigMap content
	g.Expect(env.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)).To(Succeed())
	configMap.Data["components"] = testComponents + "\n# Updated shared content"
	g.Expect(env.Update(ctx, configMap)).To(Succeed())

	t.Log("Shared ConfigMap updated")

	// Wait for all providers to be reconciled with new hashes
	for i, provider := range providers {
		g.Eventually(func() bool {
			if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
				return false
			}

			currentHash := provider.GetAnnotations()[appliedSpecHashAnnotation]
			initialHash := initialHashes[provider.GetName()]
			return currentHash != "" && currentHash != initialHash
		}, 30*time.Second).Should(BeTrue())

		t.Logf("Provider %d reconciled with new hash", i)
	}
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
