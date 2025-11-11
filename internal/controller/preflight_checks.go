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
	"errors"
	"fmt"
	"os"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	moreThanOneCoreProviderInstanceExistsMessage = "CoreProvider already exists in the cluster. Only one is allowed."
	moreThanOneProviderInstanceExistsMessage     = "There is already a %s with name %s in the cluster. Only one is allowed."
	capiVersionIncompatibilityMessage            = "CAPI operator is only compatible with %s providers, detected %s for provider %s."
	invalidGithubTokenMessage                    = "Invalid github token, please check your github token value and its permissions" //nolint:gosec
	waitingForCoreProviderReadyMessage           = "Waiting for the CoreProvider to be installed."
	incorrectCoreProviderNameMessage             = "Incorrect CoreProvider name: %s. It should be %s"
	unsupportedProviderDowngradeMessage          = "Downgrade is not supported for provider %s"

	errCoreProviderWait = errors.New(waitingForCoreProviderReadyMessage)
)

// preflightChecks performs preflight checks before installing provider.
func preflightChecks(ctx context.Context, c client.Client, provider genericprovider.GenericProvider, providerList genericprovider.GenericProviderList, mapper ProviderTypeMapper, lister ProviderLister) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Performing preflight checks")

	spec := provider.GetSpec()

	if spec.Version != "" {
		// Check that the provider version is supported.
		if err := checkProviderVersion(ctx, spec.Version, provider); err != nil {
			return err
		}
	}

	// Ensure that the CoreProvider is called "cluster-api".
	if mapper(provider) == clusterctlv1.CoreProviderType {
		if provider.ProviderName() != configclient.ClusterAPIProviderName {
			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.IncorrectCoreProviderNameReason,
				Message: fmt.Sprintf(incorrectCoreProviderNameMessage, provider.ProviderName(), configclient.ClusterAPIProviderName),
			})

			return fmt.Errorf("incorrect CoreProvider name: %s, it should be %s", provider.ProviderName(), configclient.ClusterAPIProviderName)
		}
	}

	// Check that if a predefined provider is being installed, and if it's not - ensure that FetchConfig is specified.
	isPredefinedProvider, err := isPredefinedProvider(ctx, provider.ProviderName(), mapper(provider))
	if err != nil {
		return fmt.Errorf("failed to generate a list of predefined providers: %w", err)
	}

	if !isPredefinedProvider {
		if spec.FetchConfig == nil || spec.FetchConfig.Selector == nil && spec.FetchConfig.URL == "" && spec.FetchConfig.OCI == "" {
			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.FetchConfigValidationErrorReason,
				Message: "Either Selector, OCI URL or provider URL must be provided for a not predefined provider",
			})

			return fmt.Errorf("either selector, OCI URL or provider URL must be provided for a not predefined provider %s", provider.GetName())
		}
	}

	if spec.FetchConfig != nil && spec.FetchConfig.Selector != nil && spec.FetchConfig.URL != "" {
		// If FetchConfiguration is not nil, exactly one of `URL` or `Selector` must be specified.
		conditions.Set(provider, metav1.Condition{
			Type:    operatorv1.PreflightCheckCondition,
			Status:  metav1.ConditionFalse,
			Reason:  operatorv1.FetchConfigValidationErrorReason,
			Message: "Only one of Selector and URL must be provided, not both",
		})

		return fmt.Errorf("only one of Selector and URL must be provided for provider %s", provider.GetName())
	}

	// Validate that provided GitHub token works and has repository access.
	if spec.ConfigSecret != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: provider.GetSpec().ConfigSecret.Namespace, Name: provider.GetSpec().ConfigSecret.Name}

		if err := c.Get(ctx, key, secret); err != nil {
			return fmt.Errorf("failed to get providers secret: %w", err)
		}

		if token, ok := secret.Data[configclient.GitHubTokenVariable]; ok {
			githubClient := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: string(token)},
			)))
			if _, _, err := githubClient.Organizations.List(ctx, "kubernetes-sigs", nil); err != nil {
				conditions.Set(provider, metav1.Condition{
					Type:    operatorv1.PreflightCheckCondition,
					Status:  metav1.ConditionFalse,
					Reason:  operatorv1.InvalidGithubTokenReason,
					Message: invalidGithubTokenMessage,
				})

				return fmt.Errorf("failed to validate provided github token: %w", err)
			}
		}
	}

	if err := c.List(ctx, providerList); err != nil {
		return fmt.Errorf("failed to list providers: %w", err)
	}

	// Check that no more than one instance of the provider is installed.
	for _, p := range providerList.GetItems() {
		// Skip if provider in the list is the same as provider it's compared with.
		if p.GetNamespace() == provider.GetNamespace() && p.GetName() == provider.GetName() {
			continue
		}

		// CoreProvider is a singleton resource, more than one instances should not exist
		if mapper(provider) == clusterctlv1.CoreProviderType && mapper(p) == clusterctlv1.CoreProviderType {
			log.Info(moreThanOneCoreProviderInstanceExistsMessage)

			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.MoreThanOneProviderInstanceExistsReason,
				Message: moreThanOneCoreProviderInstanceExistsMessage,
			})

			return fmt.Errorf("only one instance of CoreProvider is allowed")
		}

		// For any other provider we should check that instances with similar name exist in any namespace
		if mapper(p) != clusterctlv1.CoreProviderType && p.GetName() == provider.GetName() && mapper(p) == mapper(provider) {
			message := fmt.Sprintf(moreThanOneProviderInstanceExistsMessage, p.GetName(), p.GetNamespace())
			log.Info(message)

			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.MoreThanOneProviderInstanceExistsReason,
				Message: message,
			})

			return fmt.Errorf("only one %s provider is allowed in the cluster", p.GetName())
		}
	}

	// Wait for core provider to be ready before we install other providers.
	if mapper(provider) != clusterctlv1.CoreProviderType {
		ready := false
		if err := lister(ctx, &clusterctlv1.ProviderList{}, coreProviderIsReady(&ready, mapper)); err != nil {
			return fmt.Errorf("failed to get coreProvider ready condition: %w", err)
		}

		if !ready {
			log.Info(waitingForCoreProviderReadyMessage)

			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.WaitingForCoreProviderReadyReason,
				Message: waitingForCoreProviderReadyMessage,
			})

			return errCoreProviderWait
		}
	}

	conditions.Set(provider, metav1.Condition{
		Type:    operatorv1.PreflightCheckCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "PreflightChecksPassed",
		Message: "All preflight checks passed",
	})

	log.Info("Preflight checks passed")

	return nil
}

// checkProviderVersion verifies that target and installed provider versions are correct.
func checkProviderVersion(ctx context.Context, providerVersion string, provider genericprovider.GenericProvider) error {
	log := ctrl.LoggerFrom(ctx)

	// Check that provider version contains a valid value if it's not empty.
	targetVersion, err := version.ParseSemantic(providerVersion)
	if err != nil {
		log.Info("Version contains invalid value")

		conditions.Set(provider, metav1.Condition{
			Type:    operatorv1.PreflightCheckCondition,
			Status:  metav1.ConditionFalse,
			Reason:  operatorv1.IncorrectVersionFormatReason,
			Message: err.Error(),
		})

		return fmt.Errorf("version contains invalid value for provider %q", provider.GetName())
	}

	// Cluster API doesn't support downgrades by design. We need to report that for the user.
	if provider.GetStatus().InstalledVersion != nil && *provider.GetStatus().InstalledVersion != "" {
		installedVersion, err := version.ParseSemantic(*provider.GetStatus().InstalledVersion)
		if err != nil {
			return fmt.Errorf("installed version contains invalid value for provider %q", provider.GetName())
		}

		if targetVersion.Major() < installedVersion.Major() || targetVersion.Major() == installedVersion.Major() && targetVersion.Minor() < installedVersion.Minor() {
			conditions.Set(provider, metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.UnsupportedProviderDowngradeReason,
				Message: fmt.Sprintf(unsupportedProviderDowngradeMessage, provider.GetName()),
			})

			return fmt.Errorf("downgrade is not supported for provider %q", provider.GetName())
		}
	}

	return nil
}

// coreProviderIsReady returns true if the core provider is ready.
func coreProviderIsReady(ready *bool, mapper ProviderTypeMapper) ProviderOperation {
	return func(provider operatorv1.GenericProvider) error {
		if mapper(provider) == clusterctlv1.CoreProviderType && conditions.IsTrue(provider, clusterv1.ReadyCondition) {
			*ready = true
		}

		return nil
	}
}

// ignoreCoreProviderWaitError ignores errCoreProviderWait error.
func ignoreCoreProviderWaitError(err error) error {
	if errors.Is(err, errCoreProviderWait) {
		return nil
	}

	return err
}

// isPredefinedProvider checks if a given provider is known for Cluster API.
// The list of known providers can be found here:
// https://github.com/kubernetes-sigs/cluster-api/blob/main/cmd/clusterctl/client/config/providers_client.go
func isPredefinedProvider(ctx context.Context, providerName string, providerType clusterctlv1.ProviderType) (bool, error) {
	path := configPath
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		path = ""
	} else if err != nil {
		return false, err
	}

	// Initialize a client that contains predefined providers only.
	configClient, err := configclient.New(ctx, path)
	if err != nil {
		return false, err
	}

	// Try to find given provider in the predefined ones. If there is nothing, the function returns an error.
	_, err = configClient.Providers().Get(providerName, providerType)

	return err == nil, nil
}
