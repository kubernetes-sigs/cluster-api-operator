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
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

func TestReconcileDelete_RemovesFinalizer(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cluster-api",
			Namespace:  "test-ns",
			Finalizers: []string{operatorv1.ProviderFinalizer},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
	}

	// Verify the finalizer is present initially
	g.Expect(controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer)).To(BeTrue())

	// No delete phases means reconcileDelete should just remove the finalizer
	r := &GenericProviderReconciler{
		Provider:     provider,
		DeletePhases: []PhaseFn{},
	}

	result, err := r.reconcileDelete(context.Background(), provider)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.IsZero()).To(BeTrue())

	// Finalizer should be removed
	g.Expect(controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer)).To(BeFalse())
}

func TestReconcileDelete_WithFailingDeletePhase(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cluster-api",
			Namespace:  "test-ns",
			Finalizers: []string{operatorv1.ProviderFinalizer},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
	}

	expectedErr := fmt.Errorf("delete phase failed")

	r := &GenericProviderReconciler{
		Provider: provider,
		DeletePhases: []PhaseFn{
			func(ctx context.Context) (*Result, error) {
				return &Result{}, expectedErr
			},
		},
	}

	_, err := r.reconcileDelete(context.Background(), provider)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("delete phase failed"))

	// Finalizer should NOT be removed on error
	g.Expect(controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer)).To(BeTrue())
}

func TestReconcileDelete_WithPhaseError(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cluster-api",
			Namespace:  "test-ns",
			Finalizers: []string{operatorv1.ProviderFinalizer},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
	}

	// Return a PhaseError which should set a condition on the provider
	r := &GenericProviderReconciler{
		Provider: provider,
		DeletePhases: []PhaseFn{
			func(ctx context.Context) (*Result, error) {
				return &Result{}, &PhaseError{
					Err:    fmt.Errorf("component deletion failed"),
					Reason: "DeletionFailed",
					Type:   operatorv1.ProviderInstalledCondition,
				}
			},
		},
	}

	_, err := r.reconcileDelete(context.Background(), provider)
	g.Expect(err).To(HaveOccurred())

	// Finalizer should NOT be removed on error
	g.Expect(controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer)).To(BeTrue())

	// Verify condition was set on the provider
	found := false

	for _, c := range provider.GetConditions() {
		if c.Type == operatorv1.ProviderInstalledCondition {
			g.Expect(c.Status).To(Equal(metav1.ConditionFalse))
			g.Expect(c.Reason).To(Equal("DeletionFailed"))

			found = true
		}
	}

	g.Expect(found).To(BeTrue(), "expected ProviderInstalledCondition to be set")
}

func TestReconcileDelete_CompletedPhaseStopsReconciliation(t *testing.T) {
	g := NewWithT(t)

	provider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "cluster-api",
			Namespace:  "test-ns",
			Finalizers: []string{operatorv1.ProviderFinalizer},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "CoreProvider",
			APIVersion: "operator.cluster.x-k8s.io/v1alpha2",
		},
	}

	secondPhaseCalled := false

	r := &GenericProviderReconciler{
		Provider: provider,
		DeletePhases: []PhaseFn{
			func(ctx context.Context) (*Result, error) {
				return &Result{Completed: true}, nil
			},
			func(ctx context.Context) (*Result, error) {
				secondPhaseCalled = true
				return &Result{}, nil
			},
		},
	}

	result, err := r.reconcileDelete(context.Background(), provider)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.IsZero()).To(BeTrue())

	// Second phase should NOT have been called
	g.Expect(secondPhaseCalled).To(BeFalse())

	// Finalizer should still be present because Completed stops before finalizer removal
	g.Expect(controllerutil.ContainsFinalizer(provider, operatorv1.ProviderFinalizer)).To(BeTrue())
}
