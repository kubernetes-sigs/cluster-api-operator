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
	"encoding/json"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	utilyaml "sigs.k8s.io/cluster-api/util/yaml"
	"sigs.k8s.io/yaml"
)

// Patch defines an interface for applying patches to unstructured objects.
type Patch interface {
	Apply(obj *unstructured.Unstructured) error
}

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

// ApplyGenericPatches patches a list of unstructured objects with a list of patches.
// It is similar to the above function except in the fact that the list of patches could be strategic merge patch or RFC6902 json patches.
func ApplyGenericPatches(toPatches []unstructured.Unstructured, patches []*operatorv1.Patch) ([]unstructured.Unstructured, error) {
	afterPatch := make([]unstructured.Unstructured, len(toPatches))
	copy(afterPatch, toPatches)

	for patchIdx, p := range patches {
		patchJSON, err := yaml.YAMLToJSON([]byte(p.Patch))
		if err != nil {
			return nil, fmt.Errorf("failed to convert patch YAML to JSON: %w", err)
		}

		var ls labels.Selector
		if p.Target != nil && p.Target.LabelSelector != "" {
			ls, err = labels.Parse(p.Target.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("patch %d: failed to parse label selector %q: %w", patchIdx, p.Target.LabelSelector, err)
			}
		}

		for i := range afterPatch {
			obj := &afterPatch[i]

			match := matchSelector(obj, p.Target, ls)

			if !match {
				continue
			}

			err = inferAndApplyPatchType(obj, patchJSON)
			if err != nil {
				return nil, fmt.Errorf("patch %d: failed to apply patch to %s/%s: %w", patchIdx, obj.GetNamespace(), obj.GetName(), err)
			}
		}
	}

	return afterPatch, nil
}

func inferAndApplyPatchType(obj *unstructured.Unstructured, patchByte []byte) error {
	var (
		patch          Patch
		rfc6902Patches []*RFC6902
	)

	if err := json.Unmarshal(patchByte, &rfc6902Patches); err == nil {
		patch = NewRFC6902Patch(rfc6902Patches)
		if patch == nil {
			return fmt.Errorf("rfc6902 patch is nil")
		}

		if err := patch.Apply(obj); err != nil {
			return err
		}

		return nil
	}

	var strategicMerge apiextensionsv1.JSON
	if err := json.Unmarshal(patchByte, &strategicMerge); err == nil {
		patch = NewStrategicMergePatch(&strategicMerge)
		if patch == nil {
			return fmt.Errorf("strategic merge patch is nil")
		}

		if err = patch.Apply(obj); err != nil {
			return fmt.Errorf("failed to apply strategic merge patch: %w", err)
		}

		return nil
	}

	return fmt.Errorf("unable to infer patch type")
}
