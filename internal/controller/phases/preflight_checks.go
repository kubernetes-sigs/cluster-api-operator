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

package phases

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	moreThanOneProviderInstanceExistsMessage = "There is already a %s with name %s in the cluster. Only one is allowed."
	capiVersionIncompatibilityMessage        = "CAPI operator is only compatible with %s providers, detected %s for provider %s."
	invalidGithubTokenMessage                = "Invalid github token, please check your github token value and its permissions" //nolint:gosec
	unsupportedProviderDowngradeMessage      = "Downgrade is not supported for provider %s"
)

// PreflightChecks performs preflight checks before installing provider.
func PreflightChecks[P generic.Provider](ctx context.Context, phase generic.Group[P]) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Performing preflight checks")

	spec := phase.GetProvider().GetSpec()

	if spec.Version != "" {
		// Check that the provider version is supported.
		if err := checkProviderVersion(ctx, spec.Version, phase.GetProvider()); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check that if a predefined provider is being installed, and if it's not - ensure that FetchConfig is specified.
	isPredefinedProvider, err := isPredefinedProvider(ctx, phase.GetProvider().GetName(), phase.ClusterctlProviderType())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to generate a list of predefined providers: %w", err)
	}

	switch {
	case !isPredefinedProvider && spec.FetchConfig == nil:
		conditions.Set(phase.GetProvider(), conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.FetchConfigValidationErrorReason,
			clusterv1.ConditionSeverityError,
			"Either Selector or URL must be provided for a not predefined provider",
		))

		return ctrl.Result{}, fmt.Errorf("either selector or URL must be provided for a not predefined provider %s", phase.GetProvider().GetName())
	case spec.FetchConfig != nil && (spec.FetchConfig.Selector == nil && spec.FetchConfig.URL == ""):
		conditions.Set(phase.GetProvider(), conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.FetchConfigValidationErrorReason,
			clusterv1.ConditionSeverityError,
			"Either Selector or URL must be provided for a fetchConfig",
		))

		return ctrl.Result{}, fmt.Errorf("either selector or URL must be provided for provider %s", phase.GetProvider().GetName())
	case spec.FetchConfig != nil && spec.FetchConfig.Selector != nil && spec.FetchConfig.URL != "":
		// If FetchConfiguration is not nil, exactly one of `URL` or `Selector` must be specified.
		conditions.Set(phase.GetProvider(), conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.FetchConfigValidationErrorReason,
			clusterv1.ConditionSeverityError,
			"Only one of Selector and URL must be provided, not both",
		))

		return ctrl.Result{}, fmt.Errorf("only one of Selector and URL must be provided for provider %s", phase.GetProvider().GetName())
	}

	// Validate that provided github token works and has repository access.
	if spec.ConfigSecret != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: phase.GetProvider().GetSpec().ConfigSecret.Namespace, Name: phase.GetProvider().GetSpec().ConfigSecret.Name}

		if err := phase.GetClient().Get(ctx, key, secret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to get providers secret: %w", err)
		}

		if token, ok := secret.Data[configclient.GitHubTokenVariable]; ok {
			client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: string(token)},
			)))
			if _, _, err := client.Organizations.List(ctx, "kubernetes-sigs", nil); err != nil {
				conditions.Set(phase.GetProvider(), conditions.FalseCondition(
					operatorv1.PreflightCheckCondition,
					operatorv1.InvalidGithubTokenReason,
					clusterv1.ConditionSeverityError,
					invalidGithubTokenMessage,
				))

				return ctrl.Result{}, fmt.Errorf("failed to validate provided github token: %w", err)
			}
		}
	}

	// Check that no more than one instance of the provider is installed.
	for _, p := range phase.GetProviderList().GetItems() {
		// Skip if provider in the list is the same as provider it's compared with.
		if p.GetNamespace() == phase.GetProvider().GetNamespace() && p.GetName() == phase.GetProvider().GetName() {
			continue
		}

		preflightFalseCondition := conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.MoreThanOneProviderInstanceExistsReason,
			clusterv1.ConditionSeverityError,
			"",
		)

		// For any other provider we should check that instances with similar name exist in any namespace
		if p.GetName() == phase.GetProvider().GetName() {
			preflightFalseCondition.Message = fmt.Sprintf(moreThanOneProviderInstanceExistsMessage, p.GetName(), p.GetNamespace())
			log.Info(preflightFalseCondition.Message)
			conditions.Set(phase.GetProvider(), preflightFalseCondition)

			return ctrl.Result{}, fmt.Errorf("only one %s provider is allowed in the cluster", p.GetName())
		}
	}

	conditions.Set(phase.GetProvider(), conditions.TrueCondition(operatorv1.PreflightCheckCondition))

	log.Info("Preflight checks passed")

	return ctrl.Result{}, nil
}

// checkProviderVersion verifies that target and installed provider versions are correct.
func checkProviderVersion[P generic.Provider](ctx context.Context, providerVersion string, provider P) error {
	log := ctrl.LoggerFrom(ctx)

	// Check that provider version contains a valid value if it's not empty.
	targetVersion, err := version.ParseSemantic(providerVersion)
	if err != nil {
		log.Info("Version contains invalid value")
		conditions.Set(provider, conditions.FalseCondition(
			operatorv1.PreflightCheckCondition,
			operatorv1.IncorrectVersionFormatReason,
			clusterv1.ConditionSeverityError,
			err.Error(),
		))

		return fmt.Errorf("version contains invalid value for provider %q", provider.GetName())
	}

	// Cluster API doesn't support downgrades by design. We need to report that for the user.
	if provider.GetStatus().InstalledVersion != nil && *provider.GetStatus().InstalledVersion != "" {
		installedVersion, err := version.ParseSemantic(*provider.GetStatus().InstalledVersion)
		if err != nil {
			return fmt.Errorf("installed version contains invalid value for provider %q", provider.GetName())
		}

		if targetVersion.Major() < installedVersion.Major() || targetVersion.Major() == installedVersion.Major() && targetVersion.Minor() < installedVersion.Minor() {
			conditions.Set(provider, conditions.FalseCondition(
				operatorv1.PreflightCheckCondition,
				operatorv1.UnsupportedProviderDowngradeReason,
				clusterv1.ConditionSeverityError,
				fmt.Sprintf(unsupportedProviderDowngradeMessage, provider.GetName(), configclient.ClusterAPIProviderName),
			))

			return fmt.Errorf("downgrade is not supported for provider %q", provider.GetName())
		}
	}

	return nil
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
