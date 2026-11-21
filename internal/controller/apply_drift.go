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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// objectDrifted reports whether the live (existing) object has drifted from
// the desired state, using the dry-run result (desired) as the source of truth.
// The server increments metadata.generation when the spec would change.
func objectDrifted(desired, existing *unstructured.Unstructured) bool {
	// Spec drift: generation incremented in the dry-run result.
	if desired.GetGeneration() != existing.GetGeneration() {
		return true
	}

	// Metadata drift: labels/annotations the operator manages.
	return mapDrifted(desired.GetLabels(), existing.GetLabels()) ||
		mapDrifted(desired.GetAnnotations(), existing.GetAnnotations())
}

// mapDrifted reports whether any key/value in want (the operator's desired
// map) is missing or changed in have (the live object's map).  Keys present
// only in have are ignored (they belong to other field owners).
func mapDrifted(want, have map[string]string) bool {
	for k, v := range want {
		if have[k] != v {
			return true
		}
	}

	return false
}
