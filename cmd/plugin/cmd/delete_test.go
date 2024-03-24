/*
Copyright 2024 The Kubernetes Authors.

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

package cmd

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSelectorFromProvider(t *testing.T) {
	testCases := []struct {
		name        string
		provider    string
		expected    fields.Set
		expectedErr bool
	}{
		{
			name:     "empty provider",
			provider: "",
			expected: fields.Set{},
		},
		{
			name:     "provider with name only",
			provider: "aws",
			expected: fields.Set{
				"metadata.name": "aws",
			},
		},
		{
			name:     "provider with name and namespace",
			provider: "aws:infra",
			expected: fields.Set{
				"metadata.name":      "aws",
				"metadata.namespace": "infra",
			},
		},
		{
			name:        "invalid provider format",
			provider:    "aws:infra:extra:extra",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			actual, err := selectorFromProvider(tc.provider)
			if tc.expectedErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(actual).To(Equal(tc.expected))
			}
		})
	}
}

func TestDeleteProviders(t *testing.T) {
	tests := []struct {
		name      string
		list      generic.ProviderList
		providers []generic.Provider
		selector  fields.Set
	}{{
		name: "Delete providers",
		list: &operatorv1.AddonProviderList{},
		providers: []generic.Provider{&operatorv1.AddonProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "addon",
				Namespace: "default",
			},
		}, &operatorv1.AddonProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: "default",
			},
		}},
		selector: fields.Set{"metadata.namespace": "default"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			for _, provider := range tt.providers {
				g.Expect(env.Create(ctx, provider)).To(Succeed())
			}

			_, err := deleteProviders(ctx, env, tt.list, ctrlclient.MatchingFieldsSelector{
				Selector: fields.SelectorFromSet(tt.selector),
			})
			g.Expect(err).NotTo(HaveOccurred())

			for _, genericProvider := range tt.providers {
				g.Eventually(func() error {
					return env.Get(ctx, ctrlclient.ObjectKeyFromObject(genericProvider), genericProvider)
				}, waitShort).Should(HaveOccurred())
			}
		})
	}
}
