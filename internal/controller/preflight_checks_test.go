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
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
)

func TestPreflightChecks(t *testing.T) {
	namespaceName1 := "provider-test-ns-1"
	namespaceName2 := "provider-test-ns-2"

	testCases := []struct {
		name              string
		providers         []operatorv1.GenericProvider
		providerList      genericprovider.GenericProviderList
		mapper            ProviderTypeMapper
		expectedCondition metav1.Condition
		expectedError     bool
	}{
		{
			name: "only one core provider exists, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
							FetchConfig: &operatorv1.FetchConfiguration{
								URL: "https://example.com",
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "core provider with incorrect name, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-fancy-cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
							FetchConfig: &operatorv1.FetchConfiguration{
								URL: "https://example.com",
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Reason:  operatorv1.IncorrectCoreProviderNameReason,
				Message: "Incorrect CoreProvider name: my-fancy-cluster-api. It should be cluster-api",
				Status:  metav1.ConditionFalse,
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "two core providers were created, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "core-3",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.MoreThanOneProviderInstanceExistsReason,
				Message: moreThanOneCoreProviderInstanceExistsMessage,
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name: "two core providers were created, but accepted due to custom mapper logic",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "core-3",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			mapper: func(provider operatorv1.GenericProvider) clusterctlv1.ProviderType {
				if provider.GetName() == "core-3" {
					return clusterctlv1.ProviderTypeUnknown
				}

				return clusterctlv1.CoreProviderType
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "two core providers in two different namespaces were created, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.MoreThanOneProviderInstanceExistsReason,
				Message: moreThanOneCoreProviderInstanceExistsMessage,
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name: "only one infra provider exists, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha4",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []metav1.Condition{
								{
									Type:               clusterv1.ReadyCondition,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name:          "only one infra provider exists but core provider is not ready, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha4",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []metav1.Condition{
								{
									Type:               clusterv1.ReadyCondition,
									Status:             metav1.ConditionFalse,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.WaitingForCoreProviderReadyReason,
				Message: "Waiting for the CoreProvider to be installed.",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name: "two different infra providers exist in same namespaces, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "metal3",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha4",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []metav1.Condition{
								{
									Type:               clusterv1.ReadyCondition,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name: "two different infra providers exist in different namespaces, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "metal3",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha4",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []metav1.Condition{
								{
									Type:               clusterv1.ReadyCondition,
									Status:             metav1.ConditionTrue,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name:          "two similar infra provider exist in different namespaces, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.MoreThanOneProviderInstanceExistsReason,
				Message: fmt.Sprintf(moreThanOneProviderInstanceExistsMessage, "aws", namespaceName2),
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name: "two similarly named infra provider exist in different namespaces, but custom mapper returns differentiats types, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							FetchConfig: &operatorv1.FetchConfiguration{
								URL: "https://example.com",
							},
						},
					},
				},
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName2,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			mapper: func(provider operatorv1.GenericProvider) clusterctlv1.ProviderType {
				if provider.GetNamespace() == namespaceName2 {
					return clusterctlv1.ProviderTypeUnknown
				}

				if provider.GetName() == "cluster-api" {
					return clusterctlv1.CoreProviderType
				}

				return clusterctlv1.InfrastructureProviderType
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name:          "wrong version, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "one",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.IncorrectVersionFormatReason,
				Message: "could not parse \"one\" as version",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name: "missing version, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							FetchConfig: &operatorv1.FetchConfiguration{
								URL: "https://example.com",
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "incorrect fetchConfig, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
							FetchConfig: &operatorv1.FetchConfiguration{
								URL: "https://example.com",
								Selector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"provider-components": "aws"},
								},
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionFalse,
				Reason:  operatorv1.FetchConfigValidationErrorReason,
				Message: "Only one of Selector and URL must be provided, not both",
			},
			providerList: &operatorv1.InfrastructureProviderList{},
		},
		{
			name: "predefined core provider without fetch config, preflight check passed",
			providers: []operatorv1.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-api",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "CoreProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Status:  metav1.ConditionTrue,
				Reason:  "PreflightChecksPassed",
				Message: "All preflight checks passed",
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "custom Infrastructure Provider without fetch config, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-custom-aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Reason:  operatorv1.FetchConfigValidationErrorReason,
				Message: "Either Selector, OCI URL or provider URL must be provided for a not predefined provider",
				Status:  metav1.ConditionFalse,
			},
			providerList: &operatorv1.CoreProviderList{},
		},
		{
			name:          "custom Infrastructure Provider with fetch config with empty values, preflight check failed",
			expectedError: true,
			providers: []operatorv1.GenericProvider{
				&operatorv1.InfrastructureProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-custom-aws",
						Namespace: namespaceName1,
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "InfrastructureProvider",
						APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
					},
					Spec: operatorv1.InfrastructureProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: "v1.0.0",
							FetchConfig: &operatorv1.FetchConfiguration{
								URL:      "",
								Selector: nil,
							},
						},
					},
				},
			},
			expectedCondition: metav1.Condition{
				Type:    operatorv1.PreflightCheckCondition,
				Reason:  operatorv1.FetchConfigValidationErrorReason,
				Message: "Either Selector, OCI URL or provider URL must be provided for a not predefined provider",
				Status:  metav1.ConditionFalse,
			},
			providerList: &operatorv1.CoreProviderList{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewWithT(t)

			fakeClient := fake.NewClientBuilder().WithObjects().Build()

			for _, c := range tc.providers {
				gs.Expect(fakeClient.Create(ctx, c)).To(Succeed())
			}

			r := GenericProviderReconciler{
				Client: fakeClient,
			}

			mapper := tc.mapper
			if mapper == nil {
				mapper = util.ClusterctlProviderType
			}

			err := preflightChecks(context.Background(), fakeClient, tc.providers[0], tc.providerList, mapper, r.listProviders)
			if tc.expectedError {
				gs.Expect(err).To(HaveOccurred())
			} else {
				gs.Expect(err).ToNot(HaveOccurred())
			}

			// Check if proper condition is returned
			gs.Expect(tc.providers[0].GetStatus().Conditions).To(HaveLen(1))
			gs.Expect(tc.providers[0].GetStatus().Conditions[0].Type).To(Equal(tc.expectedCondition.Type))
			gs.Expect(tc.providers[0].GetStatus().Conditions[0].Status).To(Equal(tc.expectedCondition.Status))
			gs.Expect(tc.providers[0].GetStatus().Conditions[0].Message).To(Equal(tc.expectedCondition.Message))
		})
	}
}

func TestPreflightChecksUpgradesDowngrades(t *testing.T) {
	testCases := []struct {
		name                    string
		installedVersion        string
		targetVersion           string
		expectedConditionStatus metav1.ConditionStatus
		expectedError           bool
	}{
		{
			name:                    "upgrade core provider major version",
			expectedConditionStatus: metav1.ConditionTrue,
			installedVersion:        "v1.9.0",
			targetVersion:           "v2.0.0",
		},
		{
			name:                    "upgrade core provider minor version",
			expectedConditionStatus: metav1.ConditionTrue,
			installedVersion:        "v1.9.0",
			targetVersion:           "v1.10.0",
		},
		{
			name:                    "downgrade core provider major version",
			expectedConditionStatus: metav1.ConditionFalse,
			installedVersion:        "v2.0.0",
			targetVersion:           "v1.9.0",
			expectedError:           true,
		},
		{
			name:                    "downgrade core provider minor version",
			expectedConditionStatus: metav1.ConditionFalse,
			installedVersion:        "v1.10.0",
			targetVersion:           "v1.9.0",
			expectedError:           true,
		},
		{
			name:                    "downgrade core provider patch version",
			expectedConditionStatus: metav1.ConditionTrue,
			installedVersion:        "v1.10.1",
			targetVersion:           "v1.10.0",
		},
		{
			name:                    "same version",
			expectedConditionStatus: metav1.ConditionTrue,
			installedVersion:        "v1.10.0",
			targetVersion:           "v1.10.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewWithT(t)

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-api",
					Namespace: "provider-test-ns-1",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "CoreProvider",
					APIVersion: "operator.cluster.x-k8s.io/v1alpha1",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: tc.targetVersion,
						FetchConfig: &operatorv1.FetchConfiguration{
							URL: "https://example.com",
						},
					},
				},
				Status: operatorv1.CoreProviderStatus{
					ProviderStatus: operatorv1.ProviderStatus{
						InstalledVersion: &tc.installedVersion,
					},
				},
			}

			fakeClient := fake.NewClientBuilder().WithObjects().Build()

			gs.Expect(fakeClient.Create(ctx, provider)).To(Succeed())

			r := GenericProviderReconciler{
				Client: fakeClient,
			}

			err := preflightChecks(context.Background(), fakeClient, provider, &operatorv1.CoreProviderList{}, util.ClusterctlProviderType, r.listProviders)
			if tc.expectedError {
				gs.Expect(err).To(HaveOccurred())
			} else {
				gs.Expect(err).ToNot(HaveOccurred())
			}

			// Check if proper condition is returned
			gs.Expect(provider.GetStatus().Conditions).To(HaveLen(1))
			gs.Expect(provider.GetStatus().Conditions[0].Type).To(Equal(operatorv1.PreflightCheckCondition))
			gs.Expect(provider.GetStatus().Conditions[0].Status).To(Equal(tc.expectedConditionStatus))

			if tc.expectedConditionStatus == metav1.ConditionFalse {
				gs.Expect(provider.GetStatus().Conditions[0].Reason).To(Equal(operatorv1.UnsupportedProviderDowngradeReason))
			}
		})
	}
}
