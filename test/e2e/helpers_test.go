//go:build e2e

/*
Copyright 2022 The Kubernetes Authors.

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

package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
)

var ctx = context.Background()

const (
	operatorNamespace    = "capi-operator-system"
	cabpkSystemNamespace = "capi-kubeadm-bootstrap-system"
	cacpkSystemNamespace = "capi-kubeadm-control-plane-system"
	capiSystemNamespace  = "capi-system"
	capiOperatorRelease  = "capi-operator"

	previousCAPIVersion        = "v1.11.0"
	nextCAPIVersion            = "v1.12.0"
	coreProviderName           = configclient.ClusterAPIProviderName
	coreProviderDeploymentName = "capi-controller-manager"

	bootstrapProviderName           = "kubeadm"
	bootstrapProviderDeploymentName = "capi-kubeadm-bootstrap-controller-manager"

	cpProviderName           = "kubeadm"
	cpProviderDeploymentName = "capi-kubeadm-control-plane-controller-manager"

	infraProviderName           = "docker"
	infraProviderDeploymentName = "capd-controller-manager"

	addonProviderName           = "helm"
	addonProviderDeploymentName = "caaph-controller-manager"

	ipamProviderName           = "in-cluster"
	ipamProviderURL            = "https://github.com/kubernetes-sigs/cluster-api-ipam-provider-in-cluster/releases/latest/ipam-components.yaml"
	ipamProviderDeploymentName = "capi-ipam-in-cluster-controller-manager"

	customManifestsFolder = "resources"
	customProviderName    = "kubeadm-custom"

	// configMapMaxSize is the maximum size of a ConfigMap in bytes (1MB).
	configMapMaxSize = 1048576
)

// compressConfigMapData compresses the "components" field of a ConfigMap if it exceeds
// the maximum ConfigMap size limit. This uses gzip compression and stores the result
// in BinaryData, following the same pattern as the compressData function in
// internal/controller/manifests_downloader.go.
func compressConfigMapData(cm *corev1.ConfigMap) error {
	components, ok := cm.Data[operatorv1.ComponentsConfigMapKey]
	if !ok {
		// No components data to compress
		return nil
	}

	// Check if compression is needed
	if len(components) < configMapMaxSize {
		return nil
	}

	// Compress the data
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write([]byte(components)); err != nil {
		return fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Move compressed data to BinaryData
	if cm.BinaryData == nil {
		cm.BinaryData = make(map[string][]byte)
	}

	cm.BinaryData[operatorv1.ComponentsConfigMapKey] = buf.Bytes()
	delete(cm.Data, operatorv1.ComponentsConfigMapKey)

	// Set the compressed annotation
	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}

	cm.Annotations[operatorv1.CompressedAnnotation] = operatorv1.TrueValue

	return nil
}
