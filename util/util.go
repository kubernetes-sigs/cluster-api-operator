/*
Copyright 2021 The Kubernetes Authors.

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

package util

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
)

const (
	httpsScheme             = "https"
	githubDomain            = "github.com"
	gitlabHostPrefix        = "gitlab."
	gitlabPackagesAPIPrefix = "/api/v4/projects/"
)

func IsCoreProvider(p genericprovider.GenericProvider) bool {
	_, ok := p.(*operatorv1.CoreProvider)
	return ok
}

// ClusterctlProviderType returns the provider type from the genericProvider.
func ClusterctlProviderType(genericProvider operatorv1.GenericProvider) clusterctlv1.ProviderType {
	switch genericProvider.(type) {
	case *operatorv1.CoreProvider:
		return clusterctlv1.CoreProviderType
	case *operatorv1.ControlPlaneProvider:
		return clusterctlv1.ControlPlaneProviderType
	case *operatorv1.InfrastructureProvider:
		return clusterctlv1.InfrastructureProviderType
	case *operatorv1.BootstrapProvider:
		return clusterctlv1.BootstrapProviderType
	case *operatorv1.AddonProvider:
		return clusterctlv1.AddonProviderType
	case *operatorv1.IPAMProvider:
		return clusterctlv1.IPAMProviderType
	}

	return clusterctlv1.ProviderTypeUnknown
}

// RepositoryFactory returns the repository implementation corresponding to the provider URL.
// inspired by https://github.com/kubernetes-sigs/cluster-api/blob/124d9be7035e492f027cdc7a701b6b179451190a/cmd/clusterctl/client/repository/client.go#L170
func RepositoryFactory(ctx context.Context, providerConfig configclient.Provider, configVariablesClient configclient.VariablesClient) (repository.Repository, error) {
	// parse the repository url
	rURL, err := url.Parse(providerConfig.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository url %q", providerConfig.URL())
	}

	if rURL.Scheme != httpsScheme {
		return nil, fmt.Errorf("invalid provider url. there are no provider implementation for %q schema", rURL.Scheme)
	}

	// if the url is a GitHub repository
	if rURL.Host == githubDomain {
		repo, err := repository.NewGitHubRepository(ctx, providerConfig, configVariablesClient)
		if err != nil {
			return nil, fmt.Errorf("error creating the GitHub repository client: %w", err)
		}

		return repo, err
	}

	// if the url is a GitLab repository
	if strings.HasPrefix(rURL.Host, gitlabHostPrefix) && strings.HasPrefix(rURL.Path, gitlabPackagesAPIPrefix) {
		repo, err := repository.NewGitLabRepository(providerConfig, configVariablesClient)
		if err != nil {
			return nil, fmt.Errorf("error creating the GitLab repository client: %w", err)
		}

		return repo, err
	}

	return nil, fmt.Errorf("invalid provider url. Only GitHub and GitLab are supported for %q schema", rURL.Scheme)
}
