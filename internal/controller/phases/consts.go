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

package phases

const (
	// configPath is the path to the clusterctl config file.
	configPath = "/config/clusterctl.yaml"

	configMapVersionLabel = "provider.cluster.x-k8s.io/version"
	configMapTypeLabel    = "provider.cluster.x-k8s.io/type"
	configMapNameLabel    = "provider.cluster.x-k8s.io/name"
	operatorManagedLabel  = "managed-by.operator.cluster.x-k8s.io"

	compressedAnnotation = "provider.cluster.x-k8s.io/compressed"

	metadataConfigMapKey            = "metadata"
	componentsConfigMapKey          = "components"
	additionalManifestsConfigMapKey = "manifests"

	maxConfigMapSize = 1 * 1024 * 1024
)
