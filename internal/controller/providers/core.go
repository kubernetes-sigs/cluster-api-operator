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

package providers

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
)

type CoreProviderReconciler struct {
	generic.ProviderReconciler[*operatorv1.CoreProvider]
}

func NewCoreProviderReconcier(conn generic.Connector) generic.ProviderReconciler[*operatorv1.CoreProvider] {
	return &CoreProviderReconciler{
		ProviderReconciler: NewGenericProviderReconcier[*operatorv1.CoreProvider](conn),
	}
}

// ReconcileNormal implements GenericReconciler.
func (r *CoreProviderReconciler) PreflightChecks(
	ctx context.Context,
	provider *operatorv1.CoreProvider,
) []generic.ReconcileFn[*operatorv1.CoreProvider, generic.Group[*operatorv1.CoreProvider]] {
	return append(
		generic.NewReconcileFnList(r.corePreflightChecks),
		r.ProviderReconciler.PreflightChecks(ctx, provider)...,
	)
}

// ClusterctlProviderType returns ProviderType for the underlying clusterctl provider
func (r *CoreProviderReconciler) ClusterctlProviderType() clusterctlv1.ProviderType {
	return clusterctlv1.CoreProviderType
}

// ClusterctlProvider returns Provider stucture of the underlying clusterctl provider
func (r *CoreProviderReconciler) ClusterctlProvider(provider *operatorv1.CoreProvider) *clusterctlv1.Provider {
	clusterctlProvider := &clusterctlv1.Provider{ObjectMeta: metav1.ObjectMeta{
		Name:      provider.GetName(),
		Namespace: provider.GetNamespace(),
	},
		Type:         string(r.ClusterctlProviderType()),
		ProviderName: provider.GetName(),
		Version:      *util.Or(provider.GetStatus().InstalledVersion, ptr.To("")),
	}

	return clusterctlProvider
}

// ProviderList returns empty typed list for provider
func (r *CoreProviderReconciler) GetProviderList() generic.ProviderList {
	return &operatorv1.CoreProviderList{}
}

// corePreflightChecks performs preflight checks on core provider before installing .
func (r *CoreProviderReconciler) corePreflightChecks(ctx context.Context, phase generic.Group[*operatorv1.CoreProvider]) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Performing core provider preflight checks")

	// Ensure that the CoreProvider is called "cluster-api".
	if phase.GetProvider().GetName() != configclient.ClusterAPIProviderName {
		conditions.Set(phase.GetProvider(), conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.IncorrectCoreProviderNameReason,
			clusterv1.ConditionSeverityError,
			fmt.Sprintf(incorrectCoreProviderNameMessage, phase.GetProvider().GetName(), configclient.ClusterAPIProviderName),
		))

		return ctrl.Result{}, fmt.Errorf("incorrect CoreProvider name: %s, it should be %s", phase.GetProvider().GetName(), configclient.ClusterAPIProviderName)
	}

	// Check that no more than one instance of the provider is installed.
	if len(phase.GetProviderList().GetItems()) > 1 {
		preflightFalseCondition := conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.MoreThanOneProviderInstanceExistsReason,
			clusterv1.ConditionSeverityError,
			"",
		)

		// CoreProvider is a singleton resource, more than one instances should not exist
		log.Info(moreThanOneCoreProviderInstanceExistsMessage)
		preflightFalseCondition.Message = moreThanOneCoreProviderInstanceExistsMessage
		conditions.Set(phase.GetProvider(), preflightFalseCondition)

		return ctrl.Result{}, fmt.Errorf("only one instance of CoreProvider is allowed")

	}

	return ctrl.Result{}, nil
}
