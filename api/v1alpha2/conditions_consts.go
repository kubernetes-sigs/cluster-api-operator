/*
Copyright 2023 The Kubernetes Authors.

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

package v1alpha2

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// PreflightCheckCondition documents a Provider that has not passed preflight checks.
	PreflightCheckCondition clusterv1.ConditionType = "PreflightCheckPassed"

	// MoreThanOneProviderInstanceExistsReason (Severity=Info) documents that more than one instance of provider
	// exists in the cluster.
	MoreThanOneProviderInstanceExistsReason = "MoreThanOneExists"

	// IncorrectVersionFormatReason documents that the provider version is in the incorrect format.
	IncorrectVersionFormatReason = "IncorrectVersionFormat"

	// IncorrectCoreProviderNameReason documents that the Core provider name is incorrect.
	IncorrectCoreProviderNameReason = "IncorrectCoreProviderNameReason"

	// EmptyVersionReason documents that the provider version is in the incorrect format.
	EmptyVersionReason = "EmptyVersionReason"

	// FetchConfigValidationErrorReason documents that the FetchConfig is configured incorrectly.
	FetchConfigValidationErrorReason = "FetchConfigValidationError"

	// UnknownProviderReason documents that the provider name is not the name of a known provider.
	UnknownProviderReason = "UnknownProvider"

	// CAPIVersionIncompatibilityReason documents that the provider version is incompatible with operator.
	CAPIVersionIncompatibilityReason = "CAPIVersionIncompatibility"

	// ComponentsFetchErrorReason documents that an error occurred fetching the components.
	ComponentsFetchErrorReason = "ComponentsFetchError"

	// ComponentsUpgradeErrorReason documents that an error occurred while upgrading the components.
	ComponentsUpgradeErrorReason = "ComponentsUpgradeError"

	// OldComponentsDeletionErrorReason documents that an error occurred deleting the old components prior to upgrading.
	OldComponentsDeletionErrorReason = "OldComponentsDeletionError"

	// WaitingForCoreProviderReadyReason documents that the provider is waiting for the core provider to be ready.
	WaitingForCoreProviderReadyReason = "WaitingForCoreProviderReady"

	// InvalidGithubTokenReason documents that the provided github token is invalid.
	InvalidGithubTokenReason = "InvalidGithubTokenError"

	// NoDeploymentAvailableConditionReason documents that there is no Available condition for provider deployment yet.
	NoDeploymentAvailableConditionReason = "NoDeploymentAvailableConditionReason"

	// UnsupportedProviderDowngradeReason documents that the provider downgrade is not supported.
	UnsupportedProviderDowngradeReason = "UnsupportedProviderDowngradeReason"
)

const (
	// ProviderInstalledCondition documents a Provider that has been installed.
	ProviderInstalledCondition clusterv1.ConditionType = "ProviderInstalled"

	// ProviderUpgradedCondition documents a Provider that has been recently upgraded.
	ProviderUpgradedCondition clusterv1.ConditionType = "ProviderUpgraded"
)
