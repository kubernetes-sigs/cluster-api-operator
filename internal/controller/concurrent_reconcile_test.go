/*
Copyright 2026 The Kubernetes Authors.

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
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

// TestConcurrentReconcileSharedState runs two distinct providers through one
// GenericProviderReconciler instance concurrently — the exact situation an
// operator hits when run with --concurrency > 1: controller-runtime executes
// each Reconcile in its own goroutine, and every worker of the controller
// shares the reconciler's fields.
//
// This test must be run with -race. The data races it produces are the proof
// that per-resource state leaks across concurrent reconciles:
//
//  1. Reconcile() decodes the fetched object into r.Provider — a single struct
//     field shared by all workers (r.Client.Get(ctx, req.NamespacedName,
//     r.Provider)). The deferred patch helper and every phase then read and
//     mutate that same object.
//  2. The phase chain runs on a PhaseReconciler that is created per
//     reconciliation. If per-request state (repo, options, components, config
//     clients) or the provider object were shared between the workers of one
//     controller, one provider's state would leak into another's reconcile.
//  3. The healthcheck controller applies the same per-request deep-copy
//     pattern to its provider object.
func TestConcurrentReconcileSharedState(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))

	// Provider A: explicit version.
	provA := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{Namespace: "probe-a", Name: "alpha"},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.0.0",
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"probe": "alpha"}},
				},
			},
		},
	}

	// Provider B: empty version, forces auto-detection via
	// p.provider.SetSpec(spec) inside Load — a write into the shared object.
	provB := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{Namespace: "probe-b", Name: "bravo"},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"probe": "bravo"}},
				},
			},
		},
	}

	// A ready core provider so preflight checks don't stop the phase chain.
	core := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{Namespace: "probe-a", Name: "cluster-api"},
		Spec:       operatorv1.CoreProviderSpec{ProviderSpec: operatorv1.ProviderSpec{Version: "v1.0.0"}},
		Status: operatorv1.CoreProviderStatus{
			ProviderStatus: operatorv1.ProviderStatus{
				Conditions: []metav1.Condition{
					{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready", Message: "ready", LastTransitionTime: metav1.Now()},
				},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			provA, provB, core,
			probeConfigMap("probe-a", "alpha", "v1.0.0", "v1.0.0", "v1beta2", "1", "0"),
			probeConfigMap("probe-b", "bravo", "v2.0.0", "v2.0.0", "v1beta2", "2", "0"),
		).
		Build()

	// One reconciler per kind, exactly like cmd/main.go builds them. All
	// workers of this controller share its fields.
	r := &GenericProviderReconciler{
		Provider:     &operatorv1.InfrastructureProvider{},
		ProviderList: &operatorv1.InfrastructureProviderList{},
		Client:       cl,
	}

	// Production phase mechanism (per-request PhaseReconciler), minus the
	// phases that do network I/O or long waits (DownloadManifests, Store,
	// Upgrade, Install, Finalize). Every remaining phase writes per-request
	// state into objects that must be private to this reconcile.
	r.ReconcilePhases = func(p *PhaseReconciler) []PhaseFn {
		return []PhaseFn{
			p.ApplyFromCache,
			p.PreflightChecks,
			p.InitializePhaseReconciler,
			p.Load,
			p.Fetch,
			p.ReportStatus,
		}
	}

	reqA := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "probe-a", Name: "alpha"}}
	reqB := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "probe-b", Name: "bravo"}}

	for i := 0; i < 25; i++ {
		var wg sync.WaitGroup

		wg.Add(2)

		go func() {
			defer wg.Done()

			_, _ = r.Reconcile(context.Background(), reqA)
		}()
		go func() {
			defer wg.Done()

			_, _ = r.Reconcile(context.Background(), reqB)
		}()

		wg.Wait()
	}

	// Show which provider's state is left in the shared object.
	t.Logf("shared r.Provider ended up as: namespace=%q name=%q version=%q",
		r.Provider.GetNamespace(), r.Provider.GetName(), r.Provider.GetSpec().Version)

	for _, cond := range r.Provider.GetConditions() {
		t.Logf("  shared condition: %s=%s reason=%q message=%q", cond.Type, cond.Status, cond.Reason, cond.Message)
	}

	for _, req := range []reconcile.Request{reqA, reqB} {
		got := &operatorv1.InfrastructureProvider{}
		if err := cl.Get(context.Background(), req.NamespacedName, got); err != nil {
			t.Logf("get %s: %v", req.NamespacedName, err)
			continue
		}

		t.Logf("cluster object %s: finalizers=%v version=%q", req.NamespacedName, got.GetFinalizers(), got.GetSpec().Version)

		for _, cond := range got.GetConditions() {
			t.Logf("  %s condition: %s=%s reason=%q message=%q", req.Name, cond.Type, cond.Status, cond.Reason, cond.Message)
		}
	}
}

// probeConfigMap builds a custom provider config map, as prepared by
// TemplateManifestsConfigMap, for the given provider.
func probeConfigMap(namespace, providerName, version, labelVersion, contract, major, minor string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("infrastructure-%s-%s", providerName, version),
			Namespace: namespace,
			Labels: map[string]string{
				"probe":                                providerName,
				operatorv1.ConfigMapVersionLabelName:   labelVersion,
				operatorv1.ConfigMapTypeLabel:          "infrastructure",
				operatorv1.ConfigMapNameLabel:          providerName,
				"managed-by.operator.cluster.x-k8s.io": operatorv1.TrueValue,
			},
		},
		Data: map[string]string{
			"metadata": fmt.Sprintf(`
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: %[1]s
    minor: %[2]s
    contract: %[3]s
`, major, minor, contract),
			"components": fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cluster.x-k8s.io/provider: infrastructure-%s
    control-plane: controller-manager
  name: %s-controller-manager
  namespace: %s-system
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/provider: infrastructure-%s
      control-plane: controller-manager
  template:
    metadata:
      labels:
        cluster.x-k8s.io/provider: infrastructure-%s
        control-plane: controller-manager
    spec:
      containers:
      - image: gcr.io/google-samples/hello-app:1.0
        name: manager
`, providerName, providerName, providerName, namespace, providerName, providerName),
		},
	}
}
