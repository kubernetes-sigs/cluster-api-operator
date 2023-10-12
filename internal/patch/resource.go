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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "sigs.k8s.io/cluster-api/util/yaml"
	"sigs.k8s.io/yaml"
)

type resource struct {
	json        []byte
	patchedYAML []byte
	matchInfo   matchInfo
}

func (r *resource) applyMergePatch(patch mergePatch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}

	patched, err := jsonpatch.MergePatch(r.json, patch.json)
	if err != nil {
		return true, fmt.Errorf("failed to apply patch: %w", err)
	}

	r.json = patched

	return true, nil
}

func (r resource) matches(o matchInfo) bool {
	m := &r.matchInfo
	// we require kind to match, but if the patch does not specify
	// APIVersion we ignore it.
	if m.Kind != o.Kind {
		return false
	}

	// if api version not specified in patch we ignore it
	if o.APIVersion == "" && m.APIVersion != o.APIVersion {
		return false
	}

	// if both namespace and name are specified in patch we require them to match
	if o.Metadata.Namespace != "" && o.Metadata.Name != "" && m.Metadata.Namespace != o.Metadata.Namespace && m.Metadata.Name != o.Metadata.Name {
		return false
	}

	// if only name is specified in patch we require it to match(cluster scoped resources)
	if o.Metadata.Name != "" && m.Metadata.Name != o.Metadata.Name {
		return false
	}

	return true
}

func parseResources(toPatch []unstructured.Unstructured) ([]resource, error) {
	resources := []resource{}

	for _, obj := range toPatch {
		raw, err := utilyaml.FromUnstructured([]unstructured.Unstructured{obj})
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		json, err := yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("failted to parse resource: %w", err)
		}

		resources = append(resources, resource{
			json:      json,
			matchInfo: matchInfo,
		})
	}

	return resources, nil
}
