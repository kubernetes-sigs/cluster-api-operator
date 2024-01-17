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
	"k8s.io/apimachinery/pkg/fields"
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
