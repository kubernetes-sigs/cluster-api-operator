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

package patch

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/yaml"
)

// we match resources and patches on their v1 TypeMeta.
type matchInfo struct {
	Kind       string   `json:"kind,omitempty"`
	APIVersion string   `json:"apiVersion,omitempty"`
	Metadata   Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

func parseYAMLMatchInfo(raw []byte) (matchInfo, error) {
	m := matchInfo{}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return matchInfo{}, fmt.Errorf("failed to parse match info: %w", err)
	}

	return m, nil
}

func matchSelector(obj *unstructured.Unstructured, sel *operatorv1.PatchSelector, ls labels.Selector) bool {
	if sel == nil {
		return true
	}

	gvk := obj.GroupVersionKind()

	if sel.Group != "" && sel.Group != gvk.Group {
		return false
	}

	if sel.Version != "" && sel.Version != gvk.Version {
		return false
	}

	if sel.Kind != "" && sel.Kind != gvk.Kind {
		return false
	}

	if sel.Name != "" && sel.Name != obj.GetName() {
		return false
	}

	if sel.Namespace != "" && sel.Namespace != obj.GetNamespace() {
		return false
	}

	if sel.LabelSelector != "" {
		if !ls.Matches(labels.Set(obj.GetLabels())) {
			return false
		}
	}

	return true
}
