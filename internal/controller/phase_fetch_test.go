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

package controller

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAddNamespaceIfMissing(t *testing.T) {
	tests := []struct {
		name           string
		objs           []unstructured.Unstructured
		namespace      string
		expectAddition bool
	}{
		{
			name: "namespace already present",
			objs: []unstructured.Unstructured{
				{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]interface{}{"name": "existing"}}},
				{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]interface{}{"name": "test"}}},
			},
			namespace:      "test-ns",
			expectAddition: false,
		},
		{
			name: "namespace missing, should be added",
			objs: []unstructured.Unstructured{
				{Object: map[string]interface{}{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]interface{}{"name": "test"}}},
			},
			namespace:      "test-ns",
			expectAddition: true,
		},
		{
			name:           "empty objects, namespace should be added",
			objs:           []unstructured.Unstructured{},
			namespace:      "test-ns",
			expectAddition: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := addNamespaceIfMissing(tt.objs, tt.namespace)

			if tt.expectAddition {
				g.Expect(result).To(HaveLen(len(tt.objs) + 1))
				// Last element should be the Namespace
				last := result[len(result)-1]
				g.Expect(last.GetKind()).To(Equal("Namespace"))
				g.Expect(last.GetName()).To(Equal(tt.namespace))
			} else {
				g.Expect(result).To(HaveLen(len(tt.objs)))
			}
		})
	}
}
