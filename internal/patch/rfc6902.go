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

package patch

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RFC6902 defines a single RF6902 JSON Patch as defined by the https://www.rfc-editor.org/rfc/rfc6902.
type RFC6902 struct {
	Op    string                `json:"op"`
	Path  string                `json:"path"`
	Value *apiextensionsv1.JSON `json:"value"`
	// From is an optional field used in "move" and "copy" operations.
	From string `json:"from,omitempty"`
}

type rfc6902Patch struct {
	Patches []*RFC6902 `json:",inline"`
}

func NewRFC6902Patch(patches []*RFC6902) Patch {
	if len(patches) == 0 {
		return nil
	}

	return &rfc6902Patch{
		Patches: patches,
	}
}

func (r *rfc6902Patch) Apply(obj *unstructured.Unstructured) error {
	objJSON, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal object to JSON: %w", err)
	}

	patchJSON, err := json.Marshal(r.Patches)
	if err != nil {
		return fmt.Errorf("failed to marshal patch to JSON: %w", err)
	}

	p, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		return fmt.Errorf("failed to decode RFC6902 patch: %w", err)
	}

	mp, err := p.Apply(objJSON)
	if err != nil {
		return fmt.Errorf("failed to apply RFC6902 patch: %w", err)
	}

	if err := obj.UnmarshalJSON(mp); err != nil {
		return fmt.Errorf("failed to unmarshal patched JSON to object: %w", err)
	}

	return nil
}
