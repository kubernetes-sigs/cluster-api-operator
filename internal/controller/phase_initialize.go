/*
Copyright 2026 The Kubernetes Authors.

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
	"cmp"
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// initReaderVariables initializes the given reader with configuration variables from the provider's
// Spec.ConfigSecret if it is set.
func initReaderVariables(ctx context.Context, cl client.Client, reader configclient.Reader, provider genericprovider.GenericProvider) error {
	log := log.FromContext(ctx)

	// Fetch configuration variables from the secret. See API field docs for more info.
	if provider.GetSpec().ConfigSecret == nil {
		log.Info("No configuration secret was specified")

		return nil
	}

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: provider.GetSpec().ConfigSecret.Namespace, Name: provider.GetSpec().ConfigSecret.Name}

	if err := cl.Get(ctx, key, secret); err != nil {
		log.Error(err, "failed to get referenced secret")

		return err
	}

	for k, v := range secret.Data {
		reader.Set(k, string(v))
	}

	return nil
}

// InitializePhaseReconciler initializes phase reconciler.
func (p *PhaseReconciler) InitializePhaseReconciler(ctx context.Context) (*Result, error) {
	path := configPath
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		path = ""
	} else if err != nil {
		return &Result{}, err
	}

	// Initialize a client for interacting with the clusterctl configuration.
	initConfig, err := configclient.New(ctx, path)
	if err != nil {
		return &Result{}, err
	} else if path != "" {
		// Set the image and providers override client
		p.overridesClient = initConfig
	}

	overrideProviders := []configclient.Provider{}

	if p.overridesClient != nil {
		providers, err := p.overridesClient.Providers().List()
		if err != nil {
			return &Result{}, err
		}

		overrideProviders = providers
	}

	reader, err := p.secretReader(ctx, overrideProviders...)
	if err != nil {
		return &Result{}, err
	}

	// retrieves all custom providers using `FetchConfig` that aren't the current provider and adds them into MemoryReader.
	if err := p.providerLister(ctx, &clusterctlv1.ProviderList{}, loadCustomProvider(reader, p.provider, p.providerTypeMapper)); err != nil {
		return &Result{}, err
	}

	// Load provider's secret and config url.
	p.configClient, err = configclient.New(ctx, "", configclient.InjectReader(reader))
	if err != nil {
		return &Result{}, wrapPhaseError(err, "failed to load the secret reader", operatorv1.ProviderInstalledCondition)
	}

	// Get returns the configuration for the provider with a given name/type.
	// This is done using clusterctl internal API types.
	p.providerConfig, err = p.configClient.Providers().Get(p.provider.ProviderName(), p.providerTypeMapper(p.provider))
	if err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.UnknownProviderReason, operatorv1.ProviderInstalledCondition)
	}

	return &Result{}, nil
}

// secretReader use clusterctl MemoryReader structure to store the configuration variables
// that are obtained from a secret and try to set fetch url config.
func (p *PhaseReconciler) secretReader(ctx context.Context, providers ...configclient.Provider) (configclient.Reader, error) {
	log := ctrl.LoggerFrom(ctx)

	mr := configclient.NewMemoryReader()

	if err := mr.Init(ctx, ""); err != nil {
		return nil, err
	}

	// Fetch configuration variables from the secret. See API field docs for more info.
	if err := initReaderVariables(ctx, p.ctrlClient, mr, p.provider); err != nil {
		return nil, err
	}

	isCustom := true

	for _, provider := range providers {
		if _, err := mr.AddProvider(provider.Name(), provider.Type(), provider.URL()); err != nil {
			return nil, err
		}

		if provider.Type() == clusterctlv1.ProviderType(p.provider.GetType()) && provider.Name() == p.provider.ProviderName() {
			isCustom = false
		}
	}

	// If provided store fetch config url in memory reader.
	if p.provider.GetSpec().FetchConfig != nil {
		if p.provider.GetSpec().FetchConfig.URL != "" {
			log.Info("Custom fetch configuration url was provided")
			return mr.AddProvider(p.provider.ProviderName(), p.providerTypeMapper(p.provider), p.provider.GetSpec().FetchConfig.URL)
		}

		if p.provider.GetSpec().FetchConfig.Selector != nil {
			log.Info("Custom fetch configuration config map was provided")

			// To register a new provider from the config map, we need to specify a URL with a valid
			// format. However, since we're using data from a local config map, URLs are not needed.
			// As a workaround, we add a fake but well-formatted URL.
			return mr.AddProvider(p.provider.ProviderName(), p.providerTypeMapper(p.provider), fakeURL)
		}

		if isCustom && p.provider.GetSpec().FetchConfig.OCI != "" {
			return mr.AddProvider(p.provider.ProviderName(), p.providerTypeMapper(p.provider), fakeURL)
		}
	}

	return mr, nil
}

// loadCustomProvider loads the passed provider into the clusterctl configuration via the MemoryReader.
func loadCustomProvider(reader configclient.Reader, current operatorv1.GenericProvider, mapper ProviderTypeMapper) ProviderOperation {
	mr, ok := reader.(*configclient.MemoryReader)
	currProviderName := current.GetName()
	currProviderType := current.GetType()

	return func(provider operatorv1.GenericProvider) error {
		if !ok {
			return fmt.Errorf("unable to load custom provider, invalid reader passed")
		}

		if provider.GetName() == currProviderName && provider.GetType() == currProviderType || provider.GetSpec().FetchConfig == nil {
			return nil
		}

		_, err := mr.AddProvider(provider.ProviderName(), mapper(provider), cmp.Or(provider.GetSpec().FetchConfig.URL, fakeURL))

		return err
	}
}
