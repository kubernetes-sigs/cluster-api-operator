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

package webhook

import (
	"reflect"
	"testing"

	. "github.com/onsi/gomega"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

const (
	testNamespaceName  = "test-namespace"
	testNamespaceName1 = "test-namespace-1"
	testNamespaceName2 = "test-namespace-2"
)

func TestSetDefaultProviderSpec(t *testing.T) {
	testCases := []struct {
		name                 string
		providerSpec         *operatorv1.ProviderSpec
		namespace            string
		expectedProviderSpec *operatorv1.ProviderSpec
	}{
		{
			name: "shoud default secret namespace if not specified",
			providerSpec: &operatorv1.ProviderSpec{
				ConfigSecret: &operatorv1.SecretReference{
					Name: "test-secret",
				},
			},
			namespace: testNamespaceName,
			expectedProviderSpec: &operatorv1.ProviderSpec{
				ConfigSecret: &operatorv1.SecretReference{
					Name:      "test-secret",
					Namespace: testNamespaceName,
				},
			},
		},
		{
			name: "shoud not default secret namespace if specified",
			providerSpec: &operatorv1.ProviderSpec{
				ConfigSecret: &operatorv1.SecretReference{
					Name:      "test-secret",
					Namespace: testNamespaceName1,
				},
			},
			namespace: testNamespaceName2,
			expectedProviderSpec: &operatorv1.ProviderSpec{
				ConfigSecret: &operatorv1.SecretReference{
					Name:      "test-secret",
					Namespace: testNamespaceName1,
				},
			},
		},
		{
			name: "shoud default additional manifests namespace if not specified",
			providerSpec: &operatorv1.ProviderSpec{
				AdditionalManifestsRef: &operatorv1.ConfigmapReference{
					Name: "test-configmap",
				},
			},
			namespace: testNamespaceName,
			expectedProviderSpec: &operatorv1.ProviderSpec{
				AdditionalManifestsRef: &operatorv1.ConfigmapReference{
					Name:      "test-configmap",
					Namespace: testNamespaceName,
				},
			},
		},
		{
			name: "shoud not default additional manifests namespace if not specified",
			providerSpec: &operatorv1.ProviderSpec{
				AdditionalManifestsRef: &operatorv1.ConfigmapReference{
					Name:      "test-configmap",
					Namespace: testNamespaceName1,
				},
			},
			namespace: testNamespaceName2,
			expectedProviderSpec: &operatorv1.ProviderSpec{
				AdditionalManifestsRef: &operatorv1.ConfigmapReference{
					Name:      "test-configmap",
					Namespace: testNamespaceName1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := NewWithT(t)

			setDefaultProviderSpec(tc.providerSpec, tc.namespace)
			gs.Expect(reflect.DeepEqual(tc.providerSpec, tc.expectedProviderSpec)).To(BeTrue())
		})
	}
}
