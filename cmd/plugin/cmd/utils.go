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
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	// We have to specify a version here, because if we set "latest", clusterctl libs will try to fetch metadata.yaml file for the latest
	// release and fail since CAPI operator doesn't provide this file.
	capiOperatorManifestsURL = "https://github.com/kubernetes-sigs/cluster-api-operator/releases/v0.1.0/operator-components.yaml"
)

var capiOperatorLabels = map[string]string{
	clusterctlv1.ClusterctlCoreLabel: capiOperatorProviderName,
	"control-plane":                  "controller-manager",
}

var (
	ErrNotFound = fmt.Errorf("resource was not found")
	scheme      = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme))
}

// CreateKubeClient creates a kubernetes client from provided kubeconfig and kubecontext.
func CreateKubeClient(kubeconfigPath, kubeconfigContext string) (ctrlclient.Client, error) {
	// Use specified kubeconfig path and context
	loader := &clientcmd.ClientConfigLoadingRules{}
	if kubeconfigPath != "" {
		loader.ExplicitPath = kubeconfigPath
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader,
		&clientcmd.ConfigOverrides{
			CurrentContext: kubeconfigContext,
		}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading client config: %w", err)
	}

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

// GetLatestRelease returns the latest patch release.
func GetLatestRelease(ctx context.Context, repo repository.Repository) (string, error) {
	versions, err := repo.GetVersions(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get repository versions: %w", err)
	}

	// Search for the latest release according to semantic version ordering.
	// Releases with tag name that are not in semver format are ignored.
	parsedReleaseVersions := []*version.Version{}

	for _, v := range versions {
		sv, err := version.ParseSemantic(v)
		if err != nil {
			// discard releases with tags that are not a valid semantic versions (the user can point explicitly to such releases)
			continue
		}

		parsedReleaseVersions = append(parsedReleaseVersions, sv)
	}

	versionCandidates := parsedReleaseVersions

	if len(parsedReleaseVersions) == 0 {
		return "", errors.New("failed to find releases tagged with a valid semantic version number")
	}

	// Sort parsed versions by semantic version order.
	sort.SliceStable(versionCandidates, func(i, j int) bool {
		// Prioritize release versions over pre-releases. For example v1.0.0 > v2.0.0-alpha
		// If both are pre-releases, sort by semantic version order as usual.
		if versionCandidates[j].PreRelease() == "" && versionCandidates[i].PreRelease() != "" {
			return false
		}

		if versionCandidates[i].PreRelease() == "" && versionCandidates[j].PreRelease() != "" {
			return true
		}

		return versionCandidates[j].LessThan(versionCandidates[i])
	})

	// Limit the number of searchable versions by 3.
	size := 3
	if size > len(versionCandidates) {
		size = len(versionCandidates)
	}

	versionCandidates = versionCandidates[:size]

	for _, v := range versionCandidates {
		// Iterate through sorted versions and try to fetch a file from that release.
		// If it's completed successfully, we get the latest release.
		// Note: the fetched file will be cached and next time we will get it from the cache.
		versionString := "v" + v.String()

		_, err := repo.GetFile(ctx, versionString, repo.ComponentsPath())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// Ignore this version
				continue
			}

			return "", err
		}

		return versionString, nil
	}

	// If we reached this point, it means we didn't find any release.
	return "", errors.New("failed to find releases tagged with a valid semantic version number")
}

// retryWithExponentialBackoff repeats an operation until it passes or the exponential backoff times out.
func retryWithExponentialBackoff(ctx context.Context, opts wait.Backoff, operation func(ctx context.Context) error) error {
	i := 0
	if err := wait.ExponentialBackoffWithContext(ctx, opts, func(ctx context.Context) (bool, error) {
		i++
		if err := operation(ctx); err != nil {
			if i < opts.Steps {
				log.V(5).Info("Retrying with backoff", "cause", err.Error())
				return false, nil
			}

			return false, err
		}

		return true, nil
	}); err != nil {
		return fmt.Errorf("action failed after %d attempts: %w", i, err)
	}

	return nil
}

// newReadBackoff creates a new API Machinery backoff parameter set suitable for use with CLI cluster operations.
func newReadBackoff() wait.Backoff {
	// Return a exponential backoff configuration which returns durations for a total time of ~15s.
	// Example: 0, .25s, .6s, 1.2, 2.1s, 3.4s, 5.5s, 8s, 12s
	// Jitter is added as a random fraction of the duration multiplied by the jitter factor.
	return wait.Backoff{
		Duration: 250 * time.Millisecond,
		Factor:   1.5,
		Steps:    9,
		Jitter:   0.1,
	}
}
