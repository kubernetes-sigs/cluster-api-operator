/*
Copyright 2025 The Kubernetes Authors.

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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type loadOptions struct {
	coreProvider              string
	bootstrapProviders        []string
	controlPlaneProviders     []string
	infrastructureProviders   []string
	ipamProviders             []string
	runtimeExtensionProviders []string
	addonProviders            []string
	targetNamespace           string
	ociUrl                    string
	kubeconfig                string
	existing                  bool
}

var loadOpts = &loadOptions{}

var loadCmd = &cobra.Command{
	Use:     "preload",
	GroupID: groupManagement,
	Short:   "Preload providers to a management cluster",
	Long: LongDesc(`
		Preload provider manifests from an OCI image to a management cluster.

		To prepare an image you can use oras CLI: https://oras.land/docs/installation

		oras push ttl.sh/infrastructure-provider:v2.3.0 --artifact-type application/vnd.acme.config metadata.yaml:text/plain infrastructure-components.yaml:text/plain
	`),
	Example: Examples(`
		# Load CAPI operator manifests from OCI source.
		# capioperator preload -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap, from the given infrastructure provider.
		capioperator preload --infrastructure=aws -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap with a specific version of the given infrastructure provider in the default namespace.
		capioperator preload --infrastructure=aws::v2.3.0 -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap with a specific namespace and the latest version of the given infrastructure provider.
		capioperator preload --infrastructure=aws:custom-namespace -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap with a specific version and namespace of the given infrastructure provider.
		capioperator preload --infrastructure=aws:custom-namespace:v2.3.0 -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap with multiple infrastructure providers.
		capioperator preload --infrastructure=aws --infrastructure=vsphere -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap with a custom target namespace for the operator.
		capioperator preload --infrastructure aws --target-namespace foo -u ttl.sh/infrastructure-provider`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPreLoad()
	},
}

func init() {
	loadCmd.Flags().StringVar(&loadOpts.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file for the source management cluster. If unspecified, default discovery rules apply.")
	loadCmd.Flags().BoolVarP(&loadOpts.existing, "existing", "e", false,
		"Perform discovery on all providers in the cluster and prepare components ConfigMap for each of them.")
	loadCmd.PersistentFlags().StringVar(&loadOpts.coreProvider, "core", "",
		`Core provider version (e.g. cluster-api:v1.1.5) to add to the management cluster. If unspecified, Cluster API's latest release is used.`)
	loadCmd.PersistentFlags().StringSliceVarP(&loadOpts.infrastructureProviders, "infrastructure", "i", []string{},
		"Infrastructure providers and versions (e.g. aws:v0.5.0) to add to the management cluster.")
	loadCmd.PersistentFlags().StringSliceVarP(&loadOpts.bootstrapProviders, "bootstrap", "b", []string{},
		"Bootstrap providers and versions (e.g. kubeadm:v1.1.5) to add to the management cluster. If unspecified, Kubeadm bootstrap provider's latest release is used.")
	loadCmd.PersistentFlags().StringSliceVarP(&loadOpts.controlPlaneProviders, "control-plane", "c", []string{},
		"Control plane providers and versions (e.g. kubeadm:v1.1.5) to add to the management cluster. If unspecified, the Kubeadm control plane provider's latest release is used.")
	loadCmd.PersistentFlags().StringSliceVar(&loadOpts.ipamProviders, "ipam", nil,
		"IPAM providers and versions (e.g. infoblox:v0.0.1) to add to the management cluster.")
	loadCmd.PersistentFlags().StringSliceVar(&loadOpts.runtimeExtensionProviders, "runtime-extension", nil,
		"Runtime extension providers and versions (e.g. my-extension:v0.0.1) to add to the management cluster.")
	loadCmd.PersistentFlags().StringSliceVar(&loadOpts.addonProviders, "addon", []string{},
		"Add-on providers and versions (e.g. helm:v0.1.0) to add to the management cluster.")
	loadCmd.Flags().StringVarP(&loadOpts.targetNamespace, "target-namespace", "n", "capi-operator-system",
		"The target namespace where the operator should be deployed. If unspecified, the 'capi-operator-system' namespace is used.")
	loadCmd.Flags().StringVarP(&loadOpts.ociUrl, "artifact-url", "u", "",
		"The URL of the OCI artifact to collect component manifests from.")

	RootCmd.AddCommand(loadCmd)
}

func runPreLoad() error {
	ctx := context.Background()

	if loadOpts.ociUrl == "" {
		return fmt.Errorf("missing configMap artifacts url")
	}

	configMaps := []*v1.ConfigMap{}

	// Load Core Provider.
	if loadOpts.coreProvider != "" {
		configMap, err := templateConfigMap(ctx, clusterctlv1.CoreProviderType, loadOpts.ociUrl, loadOpts.coreProvider, loadOpts.targetNamespace)

		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for core provider: %w", err)
		} else {
			configMaps = append(configMaps, configMap)
		}
	}

	// Load Bootstrap Providers.
	for _, bootstrapProvider := range loadOpts.bootstrapProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.BootstrapProviderType, loadOpts.ociUrl, bootstrapProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for bootstrap provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Infrastructure Providers.
	for _, infrastructureProvider := range loadOpts.infrastructureProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.InfrastructureProviderType, loadOpts.ociUrl, infrastructureProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for infrastructure provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Control Plane Providers.
	for _, controlPlaneProvider := range loadOpts.controlPlaneProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.ControlPlaneProviderType, loadOpts.ociUrl, controlPlaneProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for controlplane provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Add-on Providers.
	for _, addonProvider := range loadOpts.addonProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.AddonProviderType, loadOpts.ociUrl, addonProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for addon provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load IPAM Providers.
	for _, ipamProvider := range loadOpts.ipamProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.IPAMProviderType, loadOpts.ociUrl, ipamProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for IPAM provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Runtime Extension Providers.
	for _, runtimeExtension := range loadOpts.runtimeExtensionProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.RuntimeExtensionProviderType, loadOpts.ociUrl, runtimeExtension, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for runtime extension provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	errors := []error{}

	if !loadOpts.existing {
		for _, cm := range configMaps {
			out, err := yaml.Marshal(cm)
			if err != nil {
				return fmt.Errorf("cannot serialize provider config map: %w", err)
			}

			fmt.Printf("---\n%s", string(out))
		}

		return nil
	}

	client, err := CreateKubeClient(loadOpts.kubeconfig, "")
	if err != nil {
		return fmt.Errorf("cannot create a client: %w", err)
	}

	for _, list := range operatorv1.ProviderLists {
		maps, err := fetchProviders(ctx, client, list.(genericProviderList))
		configMaps = append(configMaps, maps...)
		errors = append(errors, err)
	}

	for _, cm := range configMaps {
		out, err := yaml.Marshal(cm)
		if err != nil {
			return fmt.Errorf("cannot serialize provider config map: %w", err)
		}

		fmt.Printf("---\n%s", string(out))
	}

	return kerrors.NewAggregate(errors)
}

func fetchProviders(ctx context.Context, cl client.Client, providerList genericProviderList) ([]*v1.ConfigMap, error) {
	configMaps := []*v1.ConfigMap{}

	if err := cl.List(ctx, providerList, client.InNamespace("")); meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
		return configMaps, nil
	} else if err != nil {
		log.Error(err, fmt.Sprintf("Unable to list providers, %#v", err))

		return configMaps, err
	}

	for _, provider := range providerList.GetItems() {
		if provider.GetSpec().FetchConfig == nil || provider.GetSpec().FetchConfig.Selector == nil {
			cm, err := providerConfigMap(ctx, provider)
			if err != nil {
				return configMaps, err
			}

			configMaps = append(configMaps, cm)
		} else if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.OCI != "" {
			cm, err := ociConfigMap(ctx, provider)
			if err != nil {
				return configMaps, err
			}

			configMaps = append(configMaps, cm)
		}
	}

	return configMaps, nil
}

func templateConfigMap(ctx context.Context, providerType clusterctlv1.ProviderType, url, providerInput, defaultNamespace string) (*v1.ConfigMap, error) {
	provider, err := templateGenericProvider(providerType, providerInput, defaultNamespace, "", "")
	if err != nil {
		return nil, err
	}

	spec := provider.GetSpec()
	spec.FetchConfig = &operatorv1.FetchConfiguration{
		OCI: url,
	}

	// User didn't set the version, try to get repository default.
	if spec.Version == "" {
		configClient, err := configclient.New(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("cannot create config client: %w", err)
		}

		providerConfig, err := configClient.Providers().Get(provider.GetName(), util.ClusterctlProviderType(provider))
		if err != nil {
			if !strings.Contains(err.Error(), "failed to get configuration") {
				return nil, err
			}
		}

		repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
		if err != nil {
			return nil, fmt.Errorf("cannot create repository: %w", err)
		}

		spec.Version = repo.DefaultVersion()
	}

	provider.SetSpec(spec)

	return ociConfigMap(ctx, provider)
}

func providerConfigMap(ctx context.Context, provider operatorv1.GenericProvider) (*v1.ConfigMap, error) {
	mr := configclient.NewMemoryReader()
	if err := mr.Init(ctx, ""); err != nil {
		return nil, fmt.Errorf("unable to init memory reader: %w", err)
	}

	// If provided store fetch config url in memory reader.
	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.URL != "" {
		_, err := mr.AddProvider(provider.GetName(), util.ClusterctlProviderType(provider), provider.GetSpec().FetchConfig.URL)
		if err != nil {
			return nil, fmt.Errorf("cannot add custom url provider: %w", err)
		}
	}

	configClient, err := configclient.New(ctx, "", configclient.InjectReader(mr))
	if err != nil {
		return nil, fmt.Errorf("cannot create config client: %w", err)
	}

	providerConfig, err := configClient.Providers().Get(provider.GetName(), util.ClusterctlProviderType(provider))
	if err != nil {
		if !strings.Contains(err.Error(), "failed to get configuration") {
			return nil, err
		}
	}

	repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
	if err != nil {
		return nil, fmt.Errorf("cannot create repository: %w", err)
	}

	metadata, err := repo.GetFile(ctx, provider.GetSpec().Version, "metadata.yaml")
	if err != nil {
		err = fmt.Errorf("failed to read metadata.yaml from the repository for provider %q: %w", provider.GetName(), err)

		return nil, err
	}

	components, err := repo.GetFile(ctx, provider.GetSpec().Version, repo.ComponentsPath())
	if err != nil {
		err = fmt.Errorf("failed to read %q from the repository for provider %q: %w", repo.ComponentsPath(), provider.GetName(), err)

		return nil, err
	}

	configMap, err := controller.TemplateManifestsConfigMap(provider, controller.ProviderLabels(provider), metadata, components, true)
	if err != nil {
		err = fmt.Errorf("failed to create config map for provider %q: %w", provider.GetName(), err)

		return nil, err
	}

	// Unset owner references due to lack of existing provider owner object
	configMap.OwnerReferences = nil

	return configMap, nil
}

func ociConfigMap(ctx context.Context, provider operatorv1.GenericProvider) (*v1.ConfigMap, error) {
	store, err := controller.FetchOCI(ctx, provider, nil)
	if err != nil {
		return nil, err
	}

	metadata, err := store.GetMetadata(provider)
	if err != nil {
		return nil, err
	}

	components, err := store.GetComponents(provider)
	if err != nil {
		return nil, err
	}

	configMap, err := controller.TemplateManifestsConfigMap(provider, controller.OCILabels(provider), metadata, components, true)
	if err != nil {
		err = fmt.Errorf("failed to create config map for provider %q: %w", provider.GetName(), err)

		return nil, err
	}

	// Unset owner references due to lack of existing provider owner object
	configMap.OwnerReferences = nil

	return configMap, nil
}
