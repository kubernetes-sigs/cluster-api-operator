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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

var capiOperatorLabels = map[string]string{
	"clusterctl.cluster.x-k8s.io/core": "capi-operator",
	"control-plane":                    "controller-manager",
}

var ErrNotFound = fmt.Errorf("resource was not found")

// CreateKubeClient creates a kubernetes client from provided kubeconfig and kubecontext.
func CreateKubeClient(kubeconfigPath, kubeconfigContext string) (ctrlclient.Client, error) {
	// Use specified kubeconfig path and context
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: kubeconfigContext,
		}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading client config: %w", err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))

	client, err := ctrlclient.New(config, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	return client, nil
}

func EnsureNamespaceExists(ctx context.Context, client ctrlclient.Client, namespace string) error {
	// Check if the namespace exists
	ns := &corev1.Namespace{}

	err := client.Get(ctx, ctrlclient.ObjectKey{Name: namespace}, ns)
	if err == nil {
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("unexpected error during namespace checking: %w", err)
	}

	// Create the namespace if it doesn't exist
	newNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if err := client.Create(ctx, newNamespace); err != nil {
		return fmt.Errorf("unable to create namespace %s: %w", namespace, err)
	}

	return nil
}

// GetDeploymentByLabels fetches deployment based on the provided labels.
func GetDeploymentByLabels(ctx context.Context, client ctrlclient.Client, labels map[string]string) (*appsv1.Deployment, error) {
	var deploymentList appsv1.DeploymentList

	// Search deployments with desired labels in all namespaces.
	if err := client.List(ctx, &deploymentList, ctrlclient.MatchingLabels(labels)); err != nil {
		return nil, fmt.Errorf("cannot get a list of deployments from the server: %w", err)
	}

	if len(deploymentList.Items) > 1 {
		return nil, fmt.Errorf("more than one deployment found for given labels %v", labels)
	}

	if len(deploymentList.Items) == 0 {
		return nil, ErrNotFound
	}

	return &deploymentList.Items[0], nil
}

// CheckDeploymentAvailability checks if the deployment with given labels is available.
func CheckDeploymentAvailability(ctx context.Context, client ctrlclient.Client, labels map[string]string) (bool, error) {
	deployment, err := GetDeploymentByLabels(ctx, client, labels)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	for _, cond := range deployment.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

// GetKubeconfigLocation will read the environment variable $KUBECONFIG otherwise set it to ~/.kube/config.
func GetKubeconfigLocation() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	return clientcmd.RecommendedHomeFile
}

func NewGenericProvider(providerType clusterctlv1.ProviderType) operatorv1.GenericProvider {
	switch providerType {
	case clusterctlv1.CoreProviderType:
		return &operatorv1.CoreProvider{}
	case clusterctlv1.BootstrapProviderType:
		return &operatorv1.BootstrapProvider{}
	case clusterctlv1.ControlPlaneProviderType:
		return &operatorv1.ControlPlaneProvider{}
	case clusterctlv1.InfrastructureProviderType:
		return &operatorv1.InfrastructureProvider{}
	case clusterctlv1.AddonProviderType:
		return &operatorv1.AddonProvider{}
	case clusterctlv1.IPAMProviderType, clusterctlv1.RuntimeExtensionProviderType, clusterctlv1.ProviderTypeUnknown:
		panic(fmt.Sprintf("unsupported provider type %s", providerType))
	default:
		panic(fmt.Sprintf("unknown provider type %s", providerType))
	}
}
