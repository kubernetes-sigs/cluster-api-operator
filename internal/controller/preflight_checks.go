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

package controller

import (
	"context"
	"fmt"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	coreProvider = "CoreProvider"
)

var (
	moreThanOneCoreProviderInstanceExistsMessage = "CoreProvider already exists in the cluster. Only one is allowed."
	moreThanOneProviderInstanceExistsMessage     = "There is already a %s with name %s in the cluster. Only one is allowed."
	capiVersionIncompatibilityMessage            = "CAPI operator is only compatible with %s providers, detected %s for provider %s."
	invalidGithubTokenMessage                    = "Invalid github token, please check your github token value and its permissions" //nolint:gosec
	waitingForCoreProviderReadyMessage           = "Waiting for the core provider to be installed."
	incorrectCoreProviderNameMessage             = "Incorrect CoreProvider name: %s. It should be %s"
)

// preflightChecks performs preflight checks before installing provider.
func preflightChecks(ctx context.Context, c client.Client, provider genericprovider.GenericProvider, providerList genericprovider.GenericProviderList) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Performing preflight checks")

	spec := provider.GetSpec()

	// Check that provider version contains a valid value if it's not empty.
	if spec.Version != "" {
		if _, err := version.ParseSemantic(spec.Version); err != nil {
			log.Info("Version contains invalid value")
			conditions.Set(provider, conditions.FalseCondition(
				operatorv1.PreflightCheckCondition,
				operatorv1.IncorrectVersionFormatReason,
				clusterv1.ConditionSeverityError,
				err.Error(),
			))

			return ctrl.Result{}, fmt.Errorf("version contains invalid value for provider %q", provider.GetName())
		}
	}

	// Ensure that the CoreProvider is called "cluster-api".
	if util.IsCoreProvider(provider) {
		if provider.GetName() != configclient.ClusterAPIProviderName {
			conditions.Set(provider, conditions.FalseCondition(
				operatorv1.PreflightCheckCondition,
				operatorv1.IncorrectCoreProviderNameReason,
				clusterv1.ConditionSeverityError,
				fmt.Sprintf(incorrectCoreProviderNameMessage, provider.GetName(), configclient.ClusterAPIProviderName),
			))

			return ctrl.Result{}, fmt.Errorf("incorrect CoreProvider name: %s, it should be %s", provider.GetName(), configclient.ClusterAPIProviderName)
		}
	}

	// Check that if a predefined provider is being installed, and if it's not - ensure that FetchConfig is specified.
	isPredefinedProvider, err := isPredefinedProvider(provider.GetName(), util.ClusterctlProviderType(provider))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate a list of predefined providers: %w", err)
	}

	if !isPredefinedProvider {
		if spec.FetchConfig == nil || spec.FetchConfig.Selector == nil && spec.FetchConfig.URL == "" {
			conditions.Set(provider, conditions.FalseCondition(
				operatorv1.PreflightCheckCondition,
				operatorv1.FetchConfigValidationErrorReason,
				clusterv1.ConditionSeverityError,
				"Either Selector or URL must be provided for a not predefined provider",
			))

			return ctrl.Result{}, fmt.Errorf("either selector or URL must be provided for a not predefined provider %s", provider.GetName())
		}
	}

	if spec.FetchConfig != nil && spec.FetchConfig.Selector != nil && spec.FetchConfig.URL != "" {
		// If FetchConfiguration is not nil, exactly one of `URL` or `Selector` must be specified.
		conditions.Set(provider, conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.FetchConfigValidationErrorReason,
			clusterv1.ConditionSeverityError,
			"Only one of Selector and URL must be provided, not both",
		))

		return ctrl.Result{}, fmt.Errorf("only one of Selector and URL must be provided for provider %s", provider.GetName())
	}

	// Validate that provided github token works and has repository access.
	if spec.ConfigSecret != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: provider.GetSpec().ConfigSecret.Namespace, Name: provider.GetSpec().ConfigSecret.Name}

		if err := c.Get(ctx, key, secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get providers secret: %w", err)
		}

		if token, ok := secret.Data[configclient.GitHubTokenVariable]; ok {
			client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: string(token)},
			)))
			if _, _, err := client.Organizations.List(ctx, "kubernetes-sigs", nil); err != nil {
				conditions.Set(provider, conditions.FalseCondition(
					operatorv1.PreflightCheckCondition,
					operatorv1.InvalidGithubTokenReason,
					clusterv1.ConditionSeverityError,
					invalidGithubTokenMessage,
				))

				return ctrl.Result{}, fmt.Errorf("failed to validate provided github token: %w", err)
			}
		}
	}

	if err := c.List(ctx, providerList.GetObject()); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list providers: %w", err)
	}

	// Check that no more than one instance of the provider is installed.
	for _, p := range providerList.GetItems() {
		// Skip if provider in the list is the same as provider it's compared with.
		if p.GetNamespace() == provider.GetNamespace() && p.GetName() == provider.GetName() {
			continue
		}

		preflightFalseCondition := conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.MoreThanOneProviderInstanceExistsReason,
			clusterv1.ConditionSeverityError,
			"",
		)

		// CoreProvider is a singleton resource, more than one instances should not exist
		if util.IsCoreProvider(p) {
			log.Info(moreThanOneCoreProviderInstanceExistsMessage)
			preflightFalseCondition.Message = moreThanOneCoreProviderInstanceExistsMessage
			conditions.Set(provider, preflightFalseCondition)

			return ctrl.Result{}, fmt.Errorf("only one instance of CoreProvider is allowed")
		}

		// For any other provider we should check that instances with similar name exist in any namespace
		if p.GetObjectKind().GroupVersionKind().Kind != coreProvider && p.GetName() == provider.GetName() {
			preflightFalseCondition.Message = fmt.Sprintf(moreThanOneProviderInstanceExistsMessage, p.GetName(), p.GetNamespace())
			log.Info(preflightFalseCondition.Message)
			conditions.Set(provider, preflightFalseCondition)

			return ctrl.Result{}, fmt.Errorf("only one %s provider is allowed in the cluster", p.GetName())
		}
	}

	// Wait for core provider to be ready before we install other providers.
	if !util.IsCoreProvider(provider) {
		ready, err := coreProviderIsReady(ctx, c)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get coreProvider ready condition: %w", err)
		}

		if !ready {
			log.Info(waitingForCoreProviderReadyMessage)
			conditions.Set(provider, conditions.FalseCondition(
				operatorv1.PreflightCheckCondition,
				operatorv1.WaitingForCoreProviderReadyReason,
				clusterv1.ConditionSeverityInfo,
				waitingForCoreProviderReadyMessage,
			))

			return ctrl.Result{RequeueAfter: preflightFailedRequeueAfter}, nil
		}
	}

	conditions.Set(provider, conditions.TrueCondition(operatorv1.PreflightCheckCondition))

	log.Info("Preflight checks passed")

	return ctrl.Result{}, nil
}

// coreProviderIsReady returns true if the core provider is ready.
func coreProviderIsReady(ctx context.Context, c client.Client) (bool, error) {
	cpl := &operatorv1.CoreProviderList{}

	if err := c.List(ctx, cpl); err != nil {
		return false, err
	}

	for _, cp := range cpl.Items {
		for _, cond := range cp.Status.Conditions {
			if cond.Type == clusterv1.ReadyCondition && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
	}

	return false, nil
}

// isPredefinedProvider checks if a given provider is known for Cluster API.
// The list of known providers can be found here:
// https://github.com/kubernetes-sigs/cluster-api/blob/main/cmd/clusterctl/client/config/providers_client.go
func isPredefinedProvider(providerName string, providerType clusterctlv1.ProviderType) (bool, error) {
	// Initialize a client that contains predefined providers only.
	configClient, err := configclient.New("")
	if err != nil {
		return false, err
	}

	// Try to find given provider in the predefined ones. If there is nothing, the function returns an error.
	_, err = configClient.Providers().Get(providerName, providerType)

	return err == nil, nil
}
