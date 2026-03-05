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

package controller

import (
	"context"
	"time"

	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// fakeURL is the stub url for custom providers, missing from clusterctl repository.
const fakeURL = "https://example.com/my-provider"

// ProviderTypeMapper is a function that maps a generic provider to a clusterctl provider type.
type ProviderTypeMapper = func(operatorv1.GenericProvider) clusterctlv1.ProviderType

// ProviderConverter is a function that maps a generic provider to a clusterctl provider.
type ProviderConverter = func(operatorv1.GenericProvider) clusterctlv1.Provider

// ProviderMapper is a function that maps a clusterctl configclient provider interface to a generic provider.
type ProviderMapper = func(ctx context.Context, provider configclient.Provider) (operatorv1.GenericProvider, error)

// ProviderOperation is a function that perform action on a generic provider.
type ProviderOperation = func(provider operatorv1.GenericProvider) error

// ProviderLister returns a list of clusterctl provider objects, and performs arbitrary operations on them.
type ProviderLister = func(ctx context.Context, list *clusterctlv1.ProviderList, ops ...ProviderOperation) error

// PhaseReconciler holds all required information for interacting with clusterctl code and
// helps to iterate through provider reconciliation phases.
type PhaseReconciler struct {
	provider           genericprovider.GenericProvider
	providerList       genericprovider.GenericProviderList
	providerMapper     ProviderMapper
	providerTypeMapper ProviderTypeMapper
	providerLister     ProviderLister
	providerConverter  ProviderConverter

	ctrlClient                 client.Client
	ctrlConfig                 *rest.Config
	repo                       repository.Repository
	contract                   string
	options                    repository.ComponentsOptions
	providerConfig             configclient.Provider
	configClient               configclient.Client
	overridesClient            configclient.Client
	components                 repository.Components
	clusterctlProvider         *clusterctlv1.Provider
	needsCompression           bool
	customAlterComponentsFuncs []repository.ComponentsAlterFn
}

// PhaseReconcilerOption is a function that configures the reconciler.
type PhaseReconcilerOption func(*PhaseReconciler)

// WithProviderTypeMapper configures the reconciler to use the given clustectlv1 provider type mapper.
func WithProviderTypeMapper(providerTypeMapper ProviderTypeMapper) PhaseReconcilerOption {
	return func(r *PhaseReconciler) {
		r.providerTypeMapper = providerTypeMapper
	}
}

// WithProviderLister configures the reconciler to use the given provider lister.
func WithProviderLister(providerLister ProviderLister) PhaseReconcilerOption {
	return func(r *PhaseReconciler) {
		r.providerLister = providerLister
	}
}

// WithProviderConverter configures the reconciler to use the given provider converter.
func WithProviderConverter(providerConverter ProviderConverter) PhaseReconcilerOption {
	return func(r *PhaseReconciler) {
		r.providerConverter = providerConverter
	}
}

// WithProviderMapper configures the reconciler to use the given provider mapper.
func WithProviderMapper(providerMapper ProviderMapper) PhaseReconcilerOption {
	return func(r *PhaseReconciler) {
		r.providerMapper = providerMapper
	}
}

// WithCustomAlterComponentsFuncs configures the reconciler to use the given custom alter components functions.
func WithCustomAlterComponentsFuncs(fns []repository.ComponentsAlterFn) PhaseReconcilerOption {
	return func(r *PhaseReconciler) {
		r.customAlterComponentsFuncs = fns
	}
}

// PhaseFn is a function that represent a phase of the reconciliation.
type PhaseFn func(context.Context) (*Result, error)

// Result holds the result and error from a reconciliation phase.
type Result struct {
	// Requeue tells the Controller to requeue the reconcile key.  Defaults to false.
	Requeue bool

	// RequeueAfter if greater than 0, tells the Controller to requeue the reconcile key after the Duration.
	// Implies that Requeue is true, there is no need to set Requeue to true at the same time as RequeueAfter.
	RequeueAfter time.Duration

	// Completed indicates if this phase finalized the reconcile process.
	Completed bool
}

func (r *Result) IsZero() bool {
	return r == nil || *r == Result{}
}

// PhaseError custom error type for phases.
type PhaseError struct {
	Reason   string
	Type     string
	Severity clusterv1.ConditionSeverity
	Err      error
}

func (p *PhaseError) Error() string {
	return p.Err.Error()
}

func wrapPhaseError(err error, reason string, condition string) error {
	if err == nil {
		return nil
	}

	return &PhaseError{
		Err:      err,
		Type:     condition,
		Reason:   reason,
		Severity: clusterv1.ConditionSeverityWarning,
	}
}

// NewPhaseReconciler returns phase reconciler for the given provider.
func NewPhaseReconciler(r GenericProviderReconciler, provider genericprovider.GenericProvider, providerList genericprovider.GenericProviderList, options ...PhaseReconcilerOption) *PhaseReconciler {
	rec := &PhaseReconciler{
		ctrlClient:         r.Client,
		ctrlConfig:         r.Config,
		clusterctlProvider: &clusterctlv1.Provider{},
		provider:           provider,
		providerList:       providerList,
		providerTypeMapper: util.ClusterctlProviderType,
		providerLister:     r.listProviders,
		providerConverter:  convertProvider,
		providerMapper:     r.providerMapper,
	}

	for _, o := range options {
		o(rec)
	}

	return rec
}

type ConfigMapRepositorySettings struct {
	repository.Repository
	additionalManifests string
	skipComponents      bool
	namespace           string
}

type ConfigMapRepositoryOption interface {
	ApplyToConfigMapRepository(*ConfigMapRepositorySettings)
}

type WithAdditionalManifests string

func (w WithAdditionalManifests) ApplyToConfigMapRepository(settings *ConfigMapRepositorySettings) {
	settings.additionalManifests = string(w)
}

type SkipComponents struct{}

func (s SkipComponents) ApplyToConfigMapRepository(settings *ConfigMapRepositorySettings) {
	settings.skipComponents = true
}

type InNamespace string

func (i InNamespace) ApplyToConfigMapRepository(settings *ConfigMapRepositorySettings) {
	settings.namespace = string(i)
}

// PreflightChecks a wrapper around the preflight checks.
func (p *PhaseReconciler) PreflightChecks(ctx context.Context) (*Result, error) {
	return &Result{}, preflightChecks(ctx, p.ctrlClient, p.provider, p.providerList, p.providerTypeMapper, p.providerLister)
}
