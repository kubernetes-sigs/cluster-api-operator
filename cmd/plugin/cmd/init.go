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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/yamlprocessor"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type initOptions struct {
	kubeconfig              string
	kubeconfigContext       string
	operatorVersion         string
	coreProvider            string
	bootstrapProviders      []string
	controlPlaneProviders   []string
	infrastructureProviders []string
	ipamProviders           []string
	// runtimeExtensionProviders []string
	addonProviders      []string
	targetNamespace     string
	configSecret        string
	waitProviders       bool
	waitProviderTimeout int
}

const (
	capiOperatorProviderName = "capi-operator"
	// We have to specify a version here, because if we set "latest", clusterctl libs will try to fetch metadata.yaml file for the latest
	// release and fail since CAPI operator doesn't provide this file.
	capiOperatorManifestsURL = "https://github.com/kubernetes-sigs/cluster-api-operator/releases/v0.1.0/operator-components.yaml"
)

var initOpts = &initOptions{}

var initCmd = &cobra.Command{
	Use:     "init",
	GroupID: groupManagement,
	Short:   "Initialize a management cluster",
	Long: LongDesc(`
		Initialize a management cluster.

		Installs Cluster API operator, core components, the kubeadm bootstrap provider,
		and the selected bootstrap and infrastructure providers.

		The management cluster must be an existing Kubernetes cluster, make sure
		to have enough privileges to install the desired components.

		Some providers require secrets to be created before running 'capioperator init'.
		Refer to the provider documentation, or use 'clusterctl config provider [name]' to get a list of required variables.

		See https://cluster-api.sigs.k8s.io and https://github.com/kubernetes-sigs/cluster-api-operator/blob/main/docs/README.md for more details.`),

	Example: Examples(`
		# Initialize CAPI operator only without installing any providers.
		# capioperator init

		# Initialize a management cluster, by installing the given infrastructure provider.
		#
		# Note: when this command is executed on an empty management cluster,
 		#       it automatically triggers the installation of the Cluster API core provider.
		capioperator init --infrastructure=aws --config-secret=capa-secret

		# Initialize a management cluster with a specific version of the given infrastructure provider in the default namespace.
		capioperator init --infrastructure=aws::v2.3.0 --config-secret=capa-secret

		# Initialize a management cluster with a specific namespace and the latest version of the given infrastructure provider.
		capioperator init --infrastructure=aws:custom-namespace --config-secret=capa-secret

		# Initialize a management cluster with a specific version and namespace of the given infrastructure provider.
		capioperator init --infrastructure=aws:custom-namespace:v2.3.0 --config-secret=capa-secret

		# Initialize a management cluster with a custom kubeconfig path and the given infrastructure provider.
		capioperator init --kubeconfig=foo.yaml --infrastructure=aws --config-secret=capa-secret

		# Initialize a management cluster with multiple infrastructure providers.
		capioperator init --infrastructure=aws --infrastructure=vsphere --config-secret=infra-secret

		# Initialize a management cluster with a custom target namespace for the operator.
		capioperator init --infrastructure aws --config-secret=capa-secret --target-namespace foo`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

var backoffOpts = wait.Backoff{
	Duration: 500 * time.Millisecond,
	Factor:   1.5,
	Steps:    10,
	Jitter:   0.4,
}

func init() {
	initCmd.PersistentFlags().StringVar(&initOpts.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig for the management cluster. If unspecified, default discovery rules apply.")
	initCmd.PersistentFlags().StringVar(&initOpts.kubeconfigContext, "kubeconfig-context", "",
		"Context to be used within the kubeconfig file. If empty, current context will be used.")
	initCmd.PersistentFlags().StringVar(&initOpts.operatorVersion, "operator-version", latestVersion,
		"CAPI Operator version (e.g. v0.7.0) to install on the management cluster. If unspecified, the latest release is used.")
	initCmd.PersistentFlags().StringVar(&initOpts.coreProvider, "core", "cluster-api",
		"Core provider version (e.g. cluster-api:v1.1.5) to add to the management cluster. If unspecified, Cluster API's latest release is used.")
	initCmd.PersistentFlags().StringSliceVarP(&initOpts.infrastructureProviders, "infrastructure", "i", []string{},
		"Infrastructure providers and versions (e.g. aws:v0.5.0) to add to the management cluster.")
	initCmd.PersistentFlags().StringSliceVarP(&initOpts.bootstrapProviders, "bootstrap", "b", []string{"kubeadm"},
		"Bootstrap providers and versions (e.g. kubeadm:v1.1.5) to add to the management cluster. If unspecified, Kubeadm bootstrap provider's latest release is used.")
	initCmd.PersistentFlags().StringSliceVarP(&initOpts.controlPlaneProviders, "control-plane", "c", []string{"kubeadm"},
		"Control plane providers and versions (e.g. kubeadm:v1.1.5) to add to the management cluster. If unspecified, the Kubeadm control plane provider's latest release is used.")
	initCmd.PersistentFlags().StringSliceVar(&initOpts.ipamProviders, "ipam", nil,
		"IPAM providers and versions (e.g. infoblox:v0.0.1) to add to the management cluster.")
	// initCmd.PersistentFlags().StringSliceVar(&initOpts.runtimeExtensionProviders, "runtime-extension", nil,
	//	"Runtime extension providers and versions (e.g. test:v0.0.1) to add to the management cluster.")
	initCmd.PersistentFlags().StringSliceVar(&initOpts.addonProviders, "addon", []string{},
		"Add-on providers and versions (e.g. helm:v0.1.0) to add to the management cluster.")
	initCmd.Flags().StringVarP(&initOpts.targetNamespace, "target-namespace", "n", "capi-operator-system",
		"The target namespace where the operator should be deployed. If unspecified, the 'capi-operator-system' namespace is used.")
	initCmd.Flags().StringVar(&initOpts.configSecret, "config-secret", "",
		"The config secret reference in format <config_secret_name>:<config_secret_namespace> to be used for the management cluster. If namespace is unspecified, target namespace will be used.")
	initCmd.Flags().BoolVar(&initOpts.waitProviders, "wait-providers", true,
		"Wait for providers to be installed.")
	initCmd.Flags().IntVar(&initOpts.waitProviderTimeout, "wait-provider-timeout", 5*60,
		"Wait timeout per provider installation in seconds. This value is ignored if --wait-providers is false")

	RootCmd.AddCommand(initCmd)
}

func runInit() error {
	ctx := context.Background()

	if initOpts.kubeconfig == "" {
		initOpts.kubeconfig = GetKubeconfigLocation()
	}

	client, err := CreateKubeClient(initOpts.kubeconfig, initOpts.kubeconfigContext)
	if err != nil {
		return fmt.Errorf("cannot create a client: %w", err)
	}

	log.Info("Checking that Cert Manager is installed and running.")

	// Ensure that cert manager is installed.
	if err := ensureCertManager(ctx, initOpts); err != nil {
		return fmt.Errorf("cannot ensure that cert manager is installed: %w", err)
	}

	log.Info("Checking that CAPI Operator is installed and running.")

	deploymentExists, err := CheckDeploymentAvailability(ctx, client, capiOperatorLabels)
	if err != nil {
		return fmt.Errorf("cannot check CAPI operator availability: %w", err)
	}

	if deploymentExists && initOpts.operatorVersion != latestVersion {
		return fmt.Errorf("cannot specify operator version when the CAPI operator is already installed")
	}

	// Deploy CAPI operator if it doesn't exist.
	if !deploymentExists {
		log.Info("Installing CAPI operator", "Version", initOpts.operatorVersion)

		if err := deployCAPIOperator(ctx, initOpts); err != nil {
			return fmt.Errorf("cannot deploy CAPI operator: %w", err)
		}

		log.Info("Waiting for CAPI Operator to be available...")

		if err := wait.ExponentialBackoff(backoffOpts, func() (bool, error) {
			return CheckDeploymentAvailability(ctx, client, capiOperatorLabels)
		}); err != nil {
			return fmt.Errorf("cannot check CAPI operator availability: %w", err)
		}

		log.Info("CAPI Operator is successfully installed.")
	} else {
		log.Info("Skipping installing CAPI Operator as it is already installed.")
	}

	return initProviders(ctx, client, initOpts)
}

func initProviders(ctx context.Context, client ctrlclient.Client, initOpts *initOptions) error {
	createdProviders := []operatorv1.GenericProvider{}

	// Parsing secret config reference
	var configSecretName, configSecretNamespace string

	secretConfigParts := strings.Split(initOpts.configSecret, ":")
	switch len(secretConfigParts) {
	case 2:
		configSecretName = secretConfigParts[0]
		configSecretNamespace = secretConfigParts[1]
	case 1:
		configSecretName = secretConfigParts[0]
		configSecretNamespace = initOpts.targetNamespace
	default:
		return fmt.Errorf("invalid secret config reference: %s", initOpts.configSecret)
	}

	// Deploy Core Provider.
	if initOpts.coreProvider != "" {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.CoreProviderType, initOpts.coreProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("cannot create core provider: %w", err)
			}
		} else {
			createdProviders = append(createdProviders, provider)
		}
	}

	// Deploy Bootstrap Providers.
	for _, bootstrapProvider := range initOpts.bootstrapProviders {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.BootstrapProviderType, bootstrapProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("cannot create bootstrap provider: %w", err)
		}

		createdProviders = append(createdProviders, provider)
	}

	// Deploy Infrastructure Providers.
	for _, infrastructureProvider := range initOpts.infrastructureProviders {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.InfrastructureProviderType, infrastructureProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("cannot create infrastructure provider: %w", err)
		}

		createdProviders = append(createdProviders, provider)
	}

	// Deploy Control Plane Providers.
	for _, controlPlaneProvider := range initOpts.controlPlaneProviders {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.ControlPlaneProviderType, controlPlaneProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("cannot create controlplane provider: %w", err)
		}

		createdProviders = append(createdProviders, provider)
	}

	// Deploy Add-on Providers.
	for _, addonProvider := range initOpts.addonProviders {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.AddonProviderType, addonProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("cannot create addon provider: %w", err)
		}

		createdProviders = append(createdProviders, provider)
	}

	// Deploy IPAM Providers.
	for _, ipamProvider := range initOpts.ipamProviders {
		provider, err := createGenericProvider(ctx, client, clusterctlv1.IPAMProviderType, ipamProvider, initOpts.targetNamespace, configSecretName, configSecretNamespace)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			}

			return fmt.Errorf("cannot create addon provider: %w", err)
		}

		createdProviders = append(createdProviders, provider)
	}

	if initOpts.waitProviders {
		var wg sync.WaitGroup

		for providerIndex := range createdProviders {
			wg.Add(1)

			go func(provider operatorv1.GenericProvider) {
				defer wg.Done()
				checkProviderReadiness(ctx, client, provider, time.Duration(initOpts.waitProviderTimeout)*time.Second)
			}(createdProviders[providerIndex])
		}

		wg.Wait()
	}

	return nil
}

func checkProviderReadiness(ctx context.Context, client ctrlclient.Client, genericProvider operatorv1.GenericProvider, timeout time.Duration) {
	log.Info("Waiting for provider to become ready", "Type", genericProvider.GetType(), "Name", genericProvider.GetName(), "Namespace", genericProvider.GetNamespace())

	pollingInterval := 500 * time.Microsecond

	// Check if the provider is ready.
	if err := wait.PollUntilContextTimeout(ctx, pollingInterval, timeout, false, func(ctx context.Context) (done bool, err error) {
		err = client.Get(ctx, ctrlclient.ObjectKeyFromObject(genericProvider), genericProvider)
		if err != nil {
			return false, fmt.Errorf("cannot get provider: %w", err)
		}

		// Checking Ready condition for the provider.
		for _, cond := range genericProvider.GetConditions() {
			if cond.Type == clusterv1.ReadyCondition && cond.Status == corev1.ConditionTrue {
				log.Info("Provider is ready", "Type", genericProvider.GetType(), "Name", genericProvider.GetName(), "Namespace", genericProvider.GetNamespace())

				return true, nil
			}
		}

		return false, nil
	}); err != nil {
		log.Error(err, "Provider is not ready", "Type", genericProvider.GetType(), "Name", genericProvider.GetName(), "Namespace", genericProvider.GetNamespace())
	}
}

func ensureCertManager(ctx context.Context, opts *initOptions) error {
	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return fmt.Errorf("cannot create config client: %w", err)
	}

	clusterKubeconfig := cluster.Kubeconfig{
		Path:    opts.kubeconfig,
		Context: opts.kubeconfigContext,
	}

	clusterClient := cluster.New(clusterKubeconfig, configClient)

	// Before installing the operator, ensure the cert-manager Webhook is in place.
	certManager := clusterClient.CertManager()
	if err := certManager.EnsureInstalled(ctx); err != nil {
		return fmt.Errorf("cannot install cert-manager Webhook: %w", err)
	}

	return nil
}

// deployCAPIOperator deploys the CAPI operator on the management cluster.
func deployCAPIOperator(ctx context.Context, opts *initOptions) error {
	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return fmt.Errorf("cannot create config client: %w", err)
	}

	providerConfig := configclient.NewProvider(capiOperatorProviderName, capiOperatorManifestsURL, clusterctlv1.ProviderTypeUnknown)

	// Reduce waiting time for the repository creation from 30 seconds to 5.
	repoCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	repo, err := util.RepositoryFactory(repoCtx, providerConfig, configClient.Variables())
	if err != nil {
		return fmt.Errorf("cannot create repository: %w", err)
	}

	if opts.operatorVersion == latestVersion {
		// Detecting the latest release by sorting all available tags and picking that last one with release.
		opts.operatorVersion, err = GetLatestRelease(ctx, repo)
		if err != nil {
			return fmt.Errorf("cannot get latest release: %w", err)
		}

		log.Info("Detected latest operator version", "Version", opts.operatorVersion)
	}

	componentsFile, err := repo.GetFile(ctx, opts.operatorVersion, repo.ComponentsPath())
	if err != nil {
		return fmt.Errorf("cannot get components file: %w", err)
	}

	options := repository.ComponentsOptions{
		TargetNamespace:     opts.targetNamespace,
		SkipTemplateProcess: false,
		Version:             opts.operatorVersion,
	}

	components, err := repository.NewComponents(repository.ComponentsInput{
		Provider:     providerConfig,
		ConfigClient: configClient,
		Processor:    yamlprocessor.NewSimpleProcessor(),
		RawYaml:      componentsFile,
		Options:      options,
	})
	if err != nil {
		return fmt.Errorf("cannot generate CAPI operator components: %w", err)
	}

	clusterKubeconfig := cluster.Kubeconfig{
		Path:    opts.kubeconfig,
		Context: opts.kubeconfigContext,
	}

	clusterClient := cluster.New(clusterKubeconfig, configClient)

	if err := clusterClient.ProviderComponents().Create(ctx, components.Objs()); err != nil {
		return fmt.Errorf("cannot create CAPI operator components: %w", err)
	}

	return nil
}

// createGenericProvider creates a generic provider.
func createGenericProvider(ctx context.Context, client ctrlclient.Client, providerType clusterctlv1.ProviderType, providerInput, defaultNamespace, configSecretName, configSecretNamespace string) (operatorv1.GenericProvider, error) {
	// Parse the provider string
	// Format is <provider-name>:<optional-namespace>:<optional-version>
	// Example: aws:capa-system:v2.1.5 -> name: aws, namespace: capa-system, version: v2.1.5
	// Example: aws -> name: aws, namespace: <defaultNamespace>, version: <latestVersion>
	// Example: aws::v2.1.5 -> name: aws, namespace: <defaultNamespace>, version: v2.1.5
	// Example: aws:capa-system -> name: aws, namespace: capa-system, version: <latestVersion>
	var name, namespace, version string

	parts := strings.Split(providerInput, ":")
	switch len(parts) {
	case 1:
		name = parts[0]
	case 2:
		name = parts[0]
		namespace = parts[1]
	case 3:
		name = parts[0]
		namespace = parts[1]
		version = parts[2]
	default:
		return nil, fmt.Errorf("invalid provider format: %s", providerInput)
	}

	if name == "" {
		return nil, fmt.Errorf("provider name can't be empty")
	}

	rec := generic.ProviderReconcilers[providerType]
	provider := rec.GenericProvider()

	// Set name and namespace
	provider.SetName(name)

	if namespace == "" {
		namespace = defaultNamespace
	}

	provider.SetNamespace(namespace)

	// Set version
	if version != "" {
		spec := provider.GetSpec()
		spec.Version = version
		provider.SetSpec(spec)
	} else {
		version = latestVersion
	}

	// Set config secret
	if configSecretName != "" {
		spec := provider.GetSpec()

		if configSecretNamespace == "" {
			configSecretNamespace = defaultNamespace
		}

		spec.ConfigSecret = &operatorv1.SecretReference{
			Name:      configSecretName,
			Namespace: configSecretNamespace,
		}
		provider.SetSpec(spec)
	}

	// Ensure that desired namespace exists
	if err := EnsureNamespaceExists(ctx, client, namespace); err != nil {
		return nil, fmt.Errorf("cannot ensure that namespace exists: %w", err)
	}

	log.Info("Installing provider", "Type", provider.GetType(), "Name", name, "Version", version, "Namespace", namespace)

	// Create the provider
	if err := wait.ExponentialBackoff(backoffOpts, func() (bool, error) {
		if err := client.Create(ctx, provider); err != nil {
			// If the provider already exists, return immediately and do not retry.
			if apierrors.IsAlreadyExists(err) {
				log.Info("Provider already exists, skipping creation", "Type", provider.GetType(), "Name", name, "Version", version, "Namespace", namespace)

				return true, err
			}

			return false, err
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("cannot create provider: %w", err)
	}

	return provider, nil
}
