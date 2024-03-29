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
