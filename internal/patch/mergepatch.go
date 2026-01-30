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

	jsonpatch "github.com/evanphx/json-patch/v5"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type mergePatch struct {
	json      []byte
	matchInfo matchInfo
}

type strategicMergePatch struct {
	Patch *apiextensionsv1.JSON `json:",inline"`
}

func NewStrategicMergePatch(patch *apiextensionsv1.JSON) Patch {
	if patch == nil {
		return nil
	}

	return &strategicMergePatch{
		Patch: patch,
	}
}

func parseMergePatches(rawPatches []string) ([]mergePatch, error) {
	patches := []mergePatch{}

	for _, patch := range rawPatches {
		matchInfo, err := parseYAMLMatchInfo([]byte(patch))
		if err != nil {
			return nil, fmt.Errorf("failed to parse patch: %w", err)
		}

		json, err := yaml.YAMLToJSON([]byte(patch))
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
		}

		patches = append(patches, mergePatch{
			json:      json,
			matchInfo: matchInfo,
		})
	}

	return patches, nil
}

func (s *strategicMergePatch) Apply(obj *unstructured.Unstructured) error {
	objJSON, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	if patched, err := jsonpatch.MergePatch(objJSON, s.Patch.Raw); err == nil {
		if err = obj.UnmarshalJSON(patched); err != nil {
			return fmt.Errorf("failed to unmarshal patched JSON to object: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to apply merge patch: %w", err)
}
