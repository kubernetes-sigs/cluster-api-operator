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
	"os"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"oras.land/oras-go/v2/registry/remote/auth"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	providercontroller "sigs.k8s.io/cluster-api-operator/internal/controller"
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
	ociURL                    string
	kubeconfig                string
	existing                  bool
}

var loadOpts = &loadOptions{}

var loadCmd = &cobra.Command{
	Use:     "preload",
	GroupID: groupManagement,
	Short:   "Preload providers to a management cluster",
	Long: LongDesc(`
		Preload provider manifests to a management cluster.

		To publish provider manifests, "capioperator publish" subcommand can be used.

		You can also use oras CLI: https://oras.land/docs/installation

		oras push ttl.sh/infrastructure-provider:v2.3.0 metadata.yaml infrastructure-components.yaml

		Alternatively, for multi-provider OCI artifact, a fully specified name can be used for both metadata and components:

		oras push ttl.sh/infrastructure-provider:tag infrastructure-docker-v1.10.0-beta.0-metadata.yaml infrastructure-docker-v1.10.0-beta.0-components.yaml
	`),
	Example: Examples(`
		# Load CAPI operator manifests from OCI source
		# capioperator preload --core cluster-api

		# Load CAPI operator manifests from any provider source in the cluster
		# capioperator preload -e

		# Prepare provider ConfigMap from OCI, from the given infrastructure provider.
		capioperator preload --infrastructure=aws -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap from OCI with a specific version of the given infrastructure provider in the default namespace.
		capioperator preload --infrastructure=aws::v2.3.0 -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap from OCI with a specific namespace and the latest version of the given infrastructure provider.
		capioperator preload --infrastructure=aws:custom-namespace -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap from OCI with a specific version and namespace of the given infrastructure provider.
		capioperator preload --infrastructure=aws:custom-namespace:v2.3.0 -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap from OCI with multiple infrastructure providers.
		capioperator preload --infrastructure=aws --infrastructure=vsphere -u ttl.sh/infrastructure-provider

		# Prepare provider ConfigMap from OCI with a custom target namespace for the operator.
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
	loadCmd.Flags().StringVarP(&loadOpts.ociURL, "artifact-url", "u", "",
		"The URL of the OCI artifact to collect component manifests from.")

	RootCmd.AddCommand(loadCmd)
}

func runPreLoad() error {
	ctx := context.Background()

	if loadOpts.ociURL == "" {
		return fmt.Errorf("missing configMap artifacts url")
	}

	configMaps := []*corev1.ConfigMap{}

	// Load Core Provider.
	if loadOpts.coreProvider != "" {
		configMap, err := templateConfigMap(ctx, clusterctlv1.CoreProviderType, loadOpts.ociURL, loadOpts.coreProvider, loadOpts.targetNamespace)

		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for core provider: %w", err)
		} else {
			configMaps = append(configMaps, configMap)
		}
	}

	// Load Bootstrap Providers.
	for _, bootstrapProvider := range loadOpts.bootstrapProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.BootstrapProviderType, loadOpts.ociURL, bootstrapProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for bootstrap provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Infrastructure Providers.
	for _, infrastructureProvider := range loadOpts.infrastructureProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.InfrastructureProviderType, loadOpts.ociURL, infrastructureProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for infrastructure provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Control Plane Providers.
	for _, controlPlaneProvider := range loadOpts.controlPlaneProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.ControlPlaneProviderType, loadOpts.ociURL, controlPlaneProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for controlplane provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Add-on Providers.
	for _, addonProvider := range loadOpts.addonProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.AddonProviderType, loadOpts.ociURL, addonProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for addon provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load IPAM Providers.
	for _, ipamProvider := range loadOpts.ipamProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.IPAMProviderType, loadOpts.ociURL, ipamProvider, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for IPAM provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	// Load Runtime Extension Providers.
	for _, runtimeExtension := range loadOpts.runtimeExtensionProviders {
		configMap, err := templateConfigMap(ctx, clusterctlv1.RuntimeExtensionProviderType, loadOpts.ociURL, runtimeExtension, loadOpts.targetNamespace)
		if err != nil {
			return fmt.Errorf("cannot prepare manifests config map for runtime extension provider: %w", err)
		}

		configMaps = append(configMaps, configMap)
	}

	if loadOpts.existing {
		client, err := CreateKubeClient(loadOpts.kubeconfig, "")
		if err != nil {
			return fmt.Errorf("cannot create a client: %w", err)
		}

		existing, err := preloadExisting(ctx, client)
		if err != nil {
			return err
		}

		configMaps = append(configMaps, existing...)
	}

	for _, cm := range configMaps {
		out, err := yaml.Marshal(cm)
		if err != nil {
			return fmt.Errorf("cannot serialize provider config map: %w", err)
		}

		fmt.Printf("---\n%s", string(out))
	}

	return nil
}

// preloadExisting uses existing cluster kubeconfig to list providers and create configmaps with components for each provider.
func preloadExisting(ctx context.Context, cl client.Client) ([]*corev1.ConfigMap, error) {
	errors := []error{}
	configMaps := []*corev1.ConfigMap{}

	for _, list := range operatorv1.ProviderLists {
		list, ok := list.(genericProviderList)
		if !ok {
			log.V(5).Info("Expected to get GenericProviderList")
			continue
		}

		list, ok = list.DeepCopyObject().(genericProviderList)
		if !ok {
			log.V(5).Info("Expected to get GenericProviderList")
			continue
		}

		maps, err := fetchProviders(ctx, cl, list)
		configMaps = append(configMaps, maps...)
		errors = append(errors, err)
	}

	return configMaps, kerrors.NewAggregate(errors)
}

func fetchProviders(ctx context.Context, cl client.Client, providerList genericProviderList) ([]*corev1.ConfigMap, error) {
	configMaps := []*corev1.ConfigMap{}

	if err := retryWithExponentialBackoff(ctx, newReadBackoff(), func(ctx context.Context) error {
		return cl.List(ctx, providerList, client.InNamespace(""))
	}); err != nil {
		log.Error(err, fmt.Sprintf("Unable to list providers, %#v", err))

		return configMaps, err
	}

	for _, provider := range providerList.GetItems() {
		if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.OCI != "" {
			cm, err := providercontroller.OCIConfigMap(ctx, provider, ociAuthentication())
			if err != nil {
				return configMaps, err
			}

			configMaps = append(configMaps, cm)
		} else if provider.GetSpec().FetchConfig == nil || provider.GetSpec().FetchConfig.Selector == nil {
			cm, err := providerConfigMap(ctx, provider)
			if err != nil {
				return configMaps, err
			}

			configMaps = append(configMaps, cm)
		}
	}

	return configMaps, nil
}

func templateConfigMap(ctx context.Context, providerType clusterctlv1.ProviderType, url, providerInput, defaultNamespace string) (*corev1.ConfigMap, error) {
	provider, err := templateGenericProvider(providerType, providerInput, defaultNamespace, "", "")
	if err != nil {
		return nil, err
	}

	spec := provider.GetSpec()
	spec.FetchConfig = &operatorv1.FetchConfiguration{
		OCIConfiguration: operatorv1.OCIConfiguration{
			OCI: url,
		},
	}
	provider.SetSpec(spec)

	if spec.Version != "" {
		return providercontroller.OCIConfigMap(ctx, provider, ociAuthentication())
	}

	// User didn't set the version, try to get repository default.
	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("cannot create config client: %w", err)
	}

	providerConfig, err := configClient.Providers().Get(provider.ProviderName(), util.ClusterctlProviderType(provider))
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

	provider.SetSpec(spec)

	return providercontroller.OCIConfigMap(ctx, provider, ociAuthentication())
}

func providerConfigMap(ctx context.Context, provider operatorv1.GenericProvider) (*corev1.ConfigMap, error) {
	mr := configclient.NewMemoryReader()
	if err := mr.Init(ctx, ""); err != nil {
		return nil, fmt.Errorf("unable to init memory reader: %w", err)
	}

	// If provided store fetch config url in memory reader.
	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.URL != "" {
		_, err := mr.AddProvider(provider.ProviderName(), util.ClusterctlProviderType(provider), provider.GetSpec().FetchConfig.URL)
		if err != nil {
			return nil, fmt.Errorf("cannot add custom url provider: %w", err)
		}
	}

	configClient, err := configclient.New(ctx, "", configclient.InjectReader(mr))
	if err != nil {
		return nil, fmt.Errorf("cannot create config client: %w", err)
	}

	providerConfig, err := configClient.Providers().Get(provider.ProviderName(), util.ClusterctlProviderType(provider))
	if err != nil {
		if !strings.Contains(err.Error(), "failed to get configuration") {
			return nil, err
		}
	}

	repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
	if err != nil {
		return nil, fmt.Errorf("cannot create repository: %w", err)
	}

	return providercontroller.RepositoryConfigMap(ctx, provider, repo)
}

// ociAuthentication returns user supplied credentials from provider variables.
func ociAuthentication() *auth.Credential {
	username := os.Getenv(providercontroller.OCIUsernameKey)
	password := os.Getenv(providercontroller.OCIPasswordKey)
	accessToken := os.Getenv(providercontroller.OCIAccessTokenKey)
	refreshToken := os.Getenv(providercontroller.OCIRefreshTokenKey)

	if username != "" || password != "" || accessToken != "" || refreshToken != "" {
		return &auth.Credential{
			Username:     username,
			Password:     password,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
	}

	return nil
}
