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
	utilyaml "sigs.k8s.io/cluster-api/util/yaml"
	"sigs.k8s.io/yaml"
)

// ApplyPatches patches a list of unstructured objects with a list of patches.
// Patches match if their kind and apiVersion match a document, with the exception
// that if the patch does not set apiVersion it will be ignored.
func ApplyPatches(toPatch []unstructured.Unstructured, patches []string) ([]unstructured.Unstructured, error) {
	resources, err := parseResources(toPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resources: %w", err)
	}

	mergePatches, err := parseMergePatches(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to parse patches: %w", err)
	}

	result := []unstructured.Unstructured{}

	for _, r := range resources {
		for _, p := range mergePatches {
			if _, err := r.applyMergePatch(p); err != nil {
				return nil, fmt.Errorf("failed to apply patch: %w", err)
			}
		}

		r.patchedYAML, err = yaml.JSONToYAML(r.json)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		patchedObj, err := utilyaml.ToUnstructured(r.patchedYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		if len(patchedObj) == 0 {
			return nil, fmt.Errorf("patched object is empty")
		}

		result = append(result, patchedObj...)
	}

	return result, nil
}
