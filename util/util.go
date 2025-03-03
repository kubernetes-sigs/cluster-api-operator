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
	"regexp"
	"strings"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	httpsScheme             = "https"
	githubDomain            = "github.com"
	gitlabHostPrefix        = "gitlab"
	gitlabPackagesAPIPrefix = "/api/v4/projects/"
)

type genericProviderList interface {
	ctrlclient.ObjectList
	operatorv1.GenericProviderList
}

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
	case *operatorv1.RuntimeExtensionProvider:
		return clusterctlv1.RuntimeExtensionProviderType
	}

	return clusterctlv1.ProviderTypeUnknown
}

// GetCustomProviders retrieves all custom providers using `FetchConfig` that aren't the current provider name / type.
func GetCustomProviders(ctx context.Context, cl ctrlclient.Client, currProvider genericprovider.GenericProvider) ([]operatorv1.GenericProvider, error) {
	customProviders := []operatorv1.GenericProvider{}
	currProviderName := currProvider.GetName()
	currProviderType := currProvider.GetType()

	for _, providerList := range operatorv1.ProviderLists {
		genProviderList, ok := providerList.(genericProviderList)
		if !ok {
			return nil, fmt.Errorf("cannot cast providers list to genericProviderList")
		}

		if err := cl.List(ctx, genProviderList); err != nil {
			return nil, fmt.Errorf("cannot get a list of providers from the server: %w", err)
		}

		genProviderListItems := genProviderList.GetItems()
		for i, provider := range genProviderListItems {
			if provider.GetName() == currProviderName && provider.GetType() == currProviderType || provider.GetSpec().FetchConfig == nil {
				continue
			}

			customProviders = append(customProviders, genProviderListItems[i])
		}
	}

	return customProviders, nil
}

// GetGenericProvider returns the first of generic providers matching the type and the name from the configclient.Provider.
func GetGenericProvider(ctx context.Context, cl ctrlclient.Client, provider configclient.Provider) (operatorv1.GenericProvider, error) {
	var list genericProviderList

	switch provider.Type() {
	case clusterctlv1.CoreProviderType:
		list = &operatorv1.CoreProviderList{}
	case clusterctlv1.ControlPlaneProviderType:
		list = &operatorv1.ControlPlaneProviderList{}
	case clusterctlv1.InfrastructureProviderType:
		list = &operatorv1.InfrastructureProviderList{}
	case clusterctlv1.BootstrapProviderType:
		list = &operatorv1.BootstrapProviderList{}
	case clusterctlv1.AddonProviderType:
		list = &operatorv1.AddonProviderList{}
	case clusterctlv1.IPAMProviderType:
		list = &operatorv1.IPAMProviderList{}
	case clusterctlv1.RuntimeExtensionProviderType:
		list = &operatorv1.RuntimeExtensionProviderList{}
	case clusterctlv1.ProviderTypeUnknown:
		return nil, fmt.Errorf("provider %s type is not supported %s", provider.Name(), provider.Type())
	}

	if err := cl.List(ctx, list); err != nil {
		return nil, err
	}

	for _, p := range list.GetItems() {
		if p.GetName() == provider.Name() {
			return p, nil
		}
	}

	return nil, fmt.Errorf("unable to find provider manifest with name %s", provider.Name())
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

	// if the url is a GitLab repository starting with gitlab- or gitlab.
	gitlabHostRegex := regexp.MustCompile(`^` + regexp.QuoteMeta(gitlabHostPrefix) + `(-.*)?\.`) // ^gitlab(-.*)?\. to match gitlab- or gitlab.
	if gitlabHostRegex.MatchString(rURL.Host) && strings.HasPrefix(rURL.Path, gitlabPackagesAPIPrefix) {
		repo, err := repository.NewGitLabRepository(providerConfig, configVariablesClient)
		if err != nil {
			return nil, fmt.Errorf("error creating the GitLab repository client: %w", err)
		}

		return repo, err
	}

	return nil, fmt.Errorf("invalid provider url. Only GitHub and GitLab are supported for %q schema", rURL.Scheme)
}
