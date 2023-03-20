//go:build e2e
// +build e2e

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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx = context.Background()
)

const (
	timeout           = 5 * time.Minute
	operatorNamespace = "capi-operator-system"

	capiVersion = "v1.4.0"

	coreProviderName           = "cluster-api"
	coreProviderDeploymentName = "capi-controller-manager"

	bootstrapProviderName           = "kubeadm"
	bootstrapProviderDeploymentName = "capi-kubeadm-bootstrap-controller-manager"

	cpProviderName           = "kubeadm"
	cpProviderDeploymentName = "capi-kubeadm-control-plane-controller-manager"

	infraProviderName           = "docker"
	infraProviderDeploymentName = "capd-controller-manager"
)

func waitForDeployment(cl client.Client, ctx context.Context, name, namespace string) (bool, error) {
	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := cl.Get(ctx, key, deployment); err != nil {
		return false, err
	}

	for _, c := range deployment.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}
