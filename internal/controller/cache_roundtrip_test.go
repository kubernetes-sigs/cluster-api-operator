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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
)

func cacheTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(operatorv1.AddToScheme(s))
	utilruntime.Must(clusterctlv1.AddToScheme(s))

	return s
}

func TestApplyFromCache_NoCacheSecret(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "cache-no-secret")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
			},
		},
	}

	r := &GenericProviderReconciler{Client: env}
	p := &PhaseReconciler{
		ctrlClient:         env,
		provider:           provider,
		providerTypeMapper: util.ClusterctlProviderType,
		providerLister:     r.listProviders,
		providerConverter:  convertProvider,
		providerMapper:     r.providerMapper,
	}

	result, err := p.ApplyFromCache(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Completed).To(BeFalse(), "should not complete when no cache exists")
}

func TestApplyFromCache_HashMismatchSkips(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "cache-hash-mismatch")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
			Annotations: map[string]string{
				appliedSpecHashAnnotation: "stale-hash",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
			},
		},
	}

	cacheName := ProviderCacheName(provider)

	// Create a cache secret with a different hash annotation
	cacheSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacheName,
			Namespace: ns.Name,
			Annotations: map[string]string{
				appliedSpecHashAnnotation: "different-hash",
			},
		},
		Data: map[string][]byte{
			"cache": []byte(`[{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"test","namespace":"test-ns"}}]`),
		},
	}
	g.Expect(env.CreateAndWait(ctx, cacheSecret)).To(Succeed())

	r := &GenericProviderReconciler{Client: env}
	p := &PhaseReconciler{
		ctrlClient:         env,
		provider:           provider,
		providerTypeMapper: util.ClusterctlProviderType,
		providerLister:     r.listProviders,
		providerConverter:  convertProvider,
		providerMapper:     r.providerMapper,
	}

	result, err := p.ApplyFromCache(ctx)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Completed).To(BeFalse(), "should skip when hash doesn't match")
}

func TestApplyManifestsFromData_Uncompressed(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "cache-uncompressed")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	manifests := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      "test-cm",
					"namespace": ns.Name,
				},
			},
		},
	}

	data, err := json.Marshal(manifests)
	g.Expect(err).ToNot(HaveOccurred())

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
			},
		},
	}

	p := &PhaseReconciler{
		ctrlClient: env,
		provider:   provider,
	}

	err = p.applyManifestsFromData(ctx, map[string][]byte{"cache": data}, false)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the ConfigMap was created via server-side apply
	cm := &corev1.ConfigMap{}
	g.Expect(env.Get(ctx, client.ObjectKey{Name: "test-cm", Namespace: ns.Name}, cm)).To(Succeed())
	g.Expect(cm.Name).To(Equal("test-cm"))
}

func TestApplyManifestsFromData_Compressed(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "cache-compressed")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	manifests := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name":      "test-cm-compressed",
					"namespace": ns.Name,
				},
			},
		},
	}

	rawData, err := json.Marshal(manifests)
	g.Expect(err).ToNot(HaveOccurred())

	// Compress the data
	var buf bytes.Buffer
	g.Expect(compressData(&buf, rawData)).To(Succeed())

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
	}

	p := &PhaseReconciler{
		ctrlClient: env,
		provider:   provider,
	}

	err = p.applyManifestsFromData(ctx, map[string][]byte{"cache": buf.Bytes()}, true)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the ConfigMap was created via server-side apply
	cm := &corev1.ConfigMap{}
	g.Expect(env.Get(ctx, client.ObjectKey{Name: "test-cm-compressed", Namespace: ns.Name}, cm)).To(Succeed())
	g.Expect(cm.Name).To(Equal("test-cm-compressed"))
}

func TestApplyManifestsFromData_InvalidJSON(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "test-ns",
		},
	}

	fakeclient := fake.NewClientBuilder().WithScheme(cacheTestScheme()).Build()

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider:   provider,
	}

	err := p.applyManifestsFromData(context.Background(), map[string][]byte{"cache": []byte("not valid json")}, false)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyManifestsFromData_InvalidCompressedData(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "test-ns",
		},
	}

	fakeclient := fake.NewClientBuilder().WithScheme(cacheTestScheme()).Build()

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider:   provider,
	}

	err := p.applyManifestsFromData(context.Background(), map[string][]byte{"cache": []byte("not gzip data")}, true)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyManifestsFromData_EmptyData(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "test-ns",
		},
	}

	fakeclient := fake.NewClientBuilder().WithScheme(cacheTestScheme()).Build()

	p := &PhaseReconciler{
		ctrlClient: fakeclient,
		provider:   provider,
	}

	// Empty map should succeed with no errors
	err := p.applyManifestsFromData(context.Background(), map[string][]byte{}, false)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestProviderHash_Deterministic(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "hash-deterministic")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
			},
		},
	}

	hash1 := sha256.New()
	g.Expect(providerHash(ctx, env, hash1, provider)).To(Succeed())
	sum1 := fmt.Sprintf("%x", hash1.Sum(nil))

	hash2 := sha256.New()
	g.Expect(providerHash(ctx, env, hash2, provider)).To(Succeed())
	sum2 := fmt.Sprintf("%x", hash2.Sum(nil))

	g.Expect(sum1).To(Equal(sum2), "hashes must be deterministic for the same provider")
}

func TestProviderHash_ChangesWithSpec(t *testing.T) {
	g := NewWithT(t)

	ns, err := env.CreateNamespace(ctx, "hash-changes")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() {
		g.Expect(env.Cleanup(ctx, ns)).To(Succeed())
	}()

	provider1 := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
			},
		},
	}

	provider2 := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: ns.Name,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v2.0.0",
			},
		},
	}

	hash1 := sha256.New()
	g.Expect(providerHash(ctx, env, hash1, provider1)).To(Succeed())
	sum1 := fmt.Sprintf("%x", hash1.Sum(nil))

	hash2 := sha256.New()
	g.Expect(providerHash(ctx, env, hash2, provider2)).To(Succeed())
	sum2 := fmt.Sprintf("%x", hash2.Sum(nil))

	g.Expect(sum1).ToNot(Equal(sum2), "hashes must differ when spec changes")
}

func TestProviderHash_ChangesWhenEmptyTolerationsAreExplicitlySet(t *testing.T) {
	g := NewWithT(t)

	cl := fake.NewClientBuilder().WithScheme(cacheTestScheme()).Build()
	providerWithoutTolerations := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "default",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Deployment: &operatorv1.DeploymentSpec{
					Replicas: ptr.To(1),
				},
			},
		},
	}

	providerWithEmptyTolerations := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: "default",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Deployment: &operatorv1.DeploymentSpec{
					Replicas:    ptr.To(1),
					Tolerations: ptr.To([]corev1.Toleration{}),
				},
			},
		},
	}

	hash1 := sha256.New()
	g.Expect(providerHash(context.Background(), cl, hash1, providerWithoutTolerations)).To(Succeed())
	sum1 := fmt.Sprintf("%x", hash1.Sum(nil))

	hash2 := sha256.New()
	g.Expect(providerHash(context.Background(), cl, hash2, providerWithEmptyTolerations)).To(Succeed())
	sum2 := fmt.Sprintf("%x", hash2.Sum(nil))

	g.Expect(sum1).ToNot(Equal(sum2), "explicit empty tolerations should change the provider hash")
}
