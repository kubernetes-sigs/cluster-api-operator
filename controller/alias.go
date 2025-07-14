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

/*
Package controller provides aliases for internal controller types and functions
to allow external users to interact with the core controller logic.
*/
package controller

import (
	providercontroller "sigs.k8s.io/cluster-api-operator/internal/controller"
	internalhealthcheck "sigs.k8s.io/cluster-api-operator/internal/controller/healthcheck"
)

// GenericProviderReconciler wraps the internal GenericProviderReconciler.
type GenericProviderReconciler = providercontroller.GenericProviderReconciler

// GenericProviderHealthCheckReconciler wraps the internal GenericProviderHealthCheckReconciler.
type GenericProviderHealthCheckReconciler = internalhealthcheck.GenericProviderHealthCheckReconciler

// PhaseFn is an alias for the internal PhaseFn type.
type PhaseFn = providercontroller.PhaseFn

// Result is an alias for the internal Result type.
type Result = providercontroller.Result

// NewPhaseReconciler is an alias for the internal NewPhaseReconciler function.
var NewPhaseReconciler = providercontroller.NewPhaseReconciler

// ProviderTypeMapper is an alias for the internal ProviderTypeMapper type.
type ProviderTypeMapper = providercontroller.ProviderTypeMapper

// WithProviderTypeMapper is an alias for the internal WithProviderTypeMapper function.
var WithProviderTypeMapper = providercontroller.WithProviderTypeMapper

// ProviderConverter is an alias for the internal ProviderConverter type.
type ProviderConverter = providercontroller.ProviderConverter

// WithProviderConverter is an alias for the internal WithProviderConverter function.
var WithProviderConverter = providercontroller.WithProviderConverter

// ProviderLister is an alias for the internal ProviderLister type.
type ProviderLister = providercontroller.ProviderLister

// ProviderOperation is an alias for the internal ProviderOperation type.
type ProviderOperation = providercontroller.ProviderOperation

// WithProviderLister is an alias for the internal WithProviderLister function.
var WithProviderLister = providercontroller.WithProviderLister

// ProviderMapper is an alias for the internal ProviderMapper type.
type ProviderMapper = providercontroller.ProviderMapper

// WithProviderMapper is an alias for the internal WithProviderMapper function.
var WithProviderMapper = providercontroller.WithProviderMapper
