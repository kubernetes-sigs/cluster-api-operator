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
	"context"

	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
)

var ctx = context.Background()

const (
	operatorNamespace     = "capi-operator-system"
	capkbSystemNamespace  = "capi-kubeadm-bootstrap-system"
	capkcpSystemNamespace = "capi-kubeadm-control-plane-system"
	capiSystemNamespace   = "capi-system"
	capiOperatorRelease   = "capi-operator"

	previousCAPIVersion        = "v1.7.7"
	nextCAPIVersion            = "v1.8.0"
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

	customManifestsFolder = "resources/"
	customProviderName    = "kubeadm-custom"
)
