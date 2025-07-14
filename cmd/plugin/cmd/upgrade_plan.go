//nolint
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
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
)

type upgradePlanOptions struct {
	kubeconfig        string
	kubeconfigContext string
}

// certManagerUpgradePlan defines the upgrade plan if cert-manager needs to be
// upgraded to a different version.
type certManagerUpgradePlan struct {
	ExternallyManaged bool
	From, To          string
	ShouldUpgrade     bool
}

// capiOperatorUpgradePlan defines the upgrade plan if CAPI operator needs to be
// upgraded to a different version.
type capiOperatorUpgradePlan struct {
	ExternallyManaged bool
	From, To          string
	ShouldUpgrade     bool
}

// upgradePlan defines a list of possible upgrade targets for a management cluster.
type upgradePlan struct {
	Contract  string
	Providers []upgradeItem
}

type providerSource string

type providerSourceType string

var (
	providerSourceTypeBuiltin   providerSourceType = "builtin"
	providerSourceTypeCustomURL providerSourceType = "custom-url"
	providerSourceTypeConfigMap providerSourceType = "config-map"
)

// upgradeItem defines a possible upgrade target for a provider in the management cluster.
type upgradeItem struct {
	Name           string
	Namespace      string
	Type           string
	Source         providerSource
	SourceType     providerSourceType
	CurrentVersion string
	NextVersion    string
}

var upgradePlanOpts = &upgradePlanOptions{}

var upgradePlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Provide a list of recommended target versions for upgrading Cluster API providers in a management cluster",
	Long: LongDesc(`
		The upgrade plan command provides a list of recommended target versions for upgrading the
        Cluster API providers in a management cluster.

		All the providers should be supporting the same API Version of Cluster API (contract) in order
        to guarantee the proper functioning of the management cluster.

		Then, for each provider, the following upgrade options are provided:
		- The latest patch release for the current API Version of Cluster API (contract).
		- The latest patch release for the next API Version of Cluster API (contract), if available.`),

	Example: Examples(`
		# Gets the recommended target versions for upgrading Cluster API providers.
		capioperator upgrade plan`),

	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgradePlan()
	},
}

func init() {
	upgradePlanCmd.Flags().StringVar(&upgradePlanOpts.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file to use for accessing the management cluster. If empty, default discovery rules apply.")
	upgradePlanCmd.Flags().StringVar(&upgradePlanOpts.kubeconfigContext, "kubeconfig-context", "",
		"Context to be used within the kubeconfig file. If empty, current context will be used.")
}

func runUpgradePlan() error {
	ctx := context.Background()

	if upgradePlanOpts.kubeconfig == "" {
		upgradePlanOpts.kubeconfig = GetKubeconfigLocation()
	}

	client, err := CreateKubeClient(upgradePlanOpts.kubeconfig, upgradePlanOpts.kubeconfigContext)
	if err != nil {
		return fmt.Errorf("cannot create a client: %w", err)
	}

	certManUpgradePlan, err := planCertManagerUpgrade(ctx, upgradePlanOpts)
	if err != nil {
		return err
	}

	if !certManUpgradePlan.ExternallyManaged {
		if certManUpgradePlan.ShouldUpgrade {
			log.Info("Cert-Manager can be upgraded", "Installed Version", certManUpgradePlan.From, "Available Version", certManUpgradePlan.To)
		} else {
			log.Info("Cert-Manager is already up to date", "Installed Version", certManUpgradePlan.From)
		}
	} else {
		log.Info("There are no managed Cert-Manager installations found")
	}

	capiOperatorUpgradePlan, err := planCAPIOperatorUpgrade(ctx, client)
	if err != nil {
		return err
	}

	if capiOperatorUpgradePlan.ShouldUpgrade {
		log.Info("CAPI operator can be upgraded", "Installed Version", capiOperatorUpgradePlan.From, "Available Version", capiOperatorUpgradePlan.To)
	} else {
		log.Info("CAPI operator is already up to date", "Installed Version", capiOperatorUpgradePlan.From)
	}

	if capiOperatorUpgradePlan.ExternallyManaged {
		log.Info("CAPI operator is not managed by the plugin and won't be modified during upgrade")
	}

	upgradePlan, err := planUpgrade(ctx, client)
	if err != nil {
		return err
	}

	if len(upgradePlan.Providers) == 0 {
		log.Info("There are no providers in the cluster. Please use capioperator init to initialize a Cluster API management cluster.")
		return nil
	}

	// ensure provider are sorted consistently (by Type, Name, Namespace).
	sortUpgradeItems(upgradePlan)

	upgradeAvailable := false

	fmt.Printf("\nLatest release available for the %s API Version of Cluster API (contract):\n\n", upgradePlan.Contract)

	w := tabwriter.NewWriter(os.Stdout, 10, 4, 3, ' ', 0)

	if _, err := fmt.Fprintln(w, "NAME\tNAMESPACE\tTYPE\tCURRENT VERSION\tNEXT VERSION"); err != nil {
		return err
	}

	for _, upgradeItem := range upgradePlan.Providers {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", upgradeItem.Name, upgradeItem.Namespace, upgradeItem.Type, upgradeItem.CurrentVersion, prettifyTargetVersion(upgradeItem.NextVersion)); err != nil {
			return err
		}

		if upgradeItem.NextVersion != "" {
			upgradeAvailable = true
		}
	}

	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Println("")

	if upgradeAvailable {
		if upgradePlan.Contract == clusterv1.GroupVersion.Version {
			fmt.Println("You can now apply the upgrade by executing the following command:")
			fmt.Println("")
			fmt.Printf("capioperator upgrade apply --contract %s\n", upgradePlan.Contract)
		} else {
			fmt.Printf("The current version of capioperator could not upgrade to %s contract (only %s supported).\n", upgradePlan.Contract, clusterv1.GroupVersion.Version)
		}
	} else {
		fmt.Println("You are already up to date!")
	}

	fmt.Println("")

	return nil
}

func planCertManagerUpgrade(ctx context.Context, opts *upgradePlanOptions) (certManagerUpgradePlan, error) {
	upgradePlan := certManagerUpgradePlan{}

	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot create config client: %w", err)
	}

	clusterKubeconfig := cluster.Kubeconfig{
		Path:    opts.kubeconfig,
		Context: opts.kubeconfigContext,
	}

	clusterClient := cluster.New(clusterKubeconfig, configClient)

	clusterctlUpgradePlan, err := clusterClient.CertManager().PlanUpgrade(ctx)
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot create upgrade plan for cert-manager: %w", err)
	}

	return certManagerUpgradePlan{
		ExternallyManaged: clusterctlUpgradePlan.ExternallyManaged,
		From:              clusterctlUpgradePlan.From,
		To:                clusterctlUpgradePlan.To,
		ShouldUpgrade:     clusterctlUpgradePlan.ShouldUpgrade,
	}, nil
}

func planCAPIOperatorUpgrade(ctx context.Context, client ctrlclient.Client) (capiOperatorUpgradePlan, error) {
	upgradePlan := capiOperatorUpgradePlan{}

	log.Info("Checking CAPI Operator version...")

	capiOperatorDeployment, err := GetDeploymentByLabels(ctx, client, capiOperatorLabels)
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot get CAPI operator deployment: %w", err)
	}

	for _, container := range capiOperatorDeployment.Spec.Template.Spec.Containers {
		if container.Name == "manager" {
			parts := strings.Split(container.Image, ":")
			upgradePlan.From = parts[len(parts)-1]

			log.Info("Found CAPI Operator deployment", "Version", upgradePlan.From)

			break
		}
	}

	log.Info("Fetching data about all CAPI Operator releases.")

	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot create config client: %w", err)
	}

	providerConfig := configclient.NewProvider(capiOperatorProviderName, capiOperatorManifestsURL, clusterctlv1.ProviderTypeUnknown)

	repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot create repository: %w", err)
	}

	latestReleasedVersion, err := GetLatestRelease(ctx, repo)
	if err != nil {
		return upgradePlan, fmt.Errorf("cannot get latest release: %w", err)
	}

	log.Info("Found latest available CAPI Operator release", "Latest Release", latestReleasedVersion)

	upgradePlan.To = latestReleasedVersion

	upgradePlan.ShouldUpgrade = upgradePlan.From != upgradePlan.To

	upgradePlan.ExternallyManaged = isCAPIOperatorExternallyManaged(capiOperatorDeployment)

	return upgradePlan, nil
}

// isCAPIOperatorExternallyManaged returns true if the CAPI operator is not managed by the plugin.
func isCAPIOperatorExternallyManaged(deployment *appsv1.Deployment) bool {
	return deployment.Labels[clusterv1.ProviderNameLabel] != capiOperatorProviderName
}

func planUpgrade(ctx context.Context, client ctrlclient.Client) (upgradePlan, error) {
	genericProviders, contract, err := getInstalledProviders(ctx, client)
	if err != nil {
		return upgradePlan{}, fmt.Errorf("cannot get installed providers: %w", err)
	}

	upgradeItems := []upgradeItem{}

	for _, genericProvider := range genericProviders {
		providerFetchSource, providerSourceType, err := getProviderFetchConfig(ctx, genericProvider)
		if err != nil {
			return upgradePlan{}, fmt.Errorf("cannot get provider fetch URL: %w", err)
		}

		// TODO: ignore configmap source type for now.
		if providerSourceType == providerSourceTypeConfigMap {
			continue
		}

		configClient, err := configclient.New(ctx, "")
		if err != nil {
			return upgradePlan{}, fmt.Errorf("cannot create config client: %w", err)
		}

		providerConfig := configclient.NewProvider(capiOperatorProviderName, string(providerFetchSource), clusterctlv1.ProviderTypeUnknown)

		repo, err := util.RepositoryFactory(ctx, providerConfig, configClient.Variables())
		if err != nil {
			return upgradePlan{}, fmt.Errorf("cannot create repository: %w", err)
		}

		upgradeItems = append(upgradeItems, upgradeItem{
			Name:           genericProvider.ProviderName(),
			Namespace:      genericProvider.GetNamespace(),
			Type:           genericProvider.GetType(),
			CurrentVersion: genericProvider.GetSpec().Version,
			NextVersion:    repo.DefaultVersion(),
			Source:         providerFetchSource,
			SourceType:     providerSourceType,
		})
	}

	return upgradePlan{Contract: contract, Providers: upgradeItems}, nil
}

func getInstalledProviders(ctx context.Context, client ctrlclient.Client) ([]operatorv1.GenericProvider, string, error) {
	// Iterate through installed providers and create a list of upgrade plans.
	genericProviders := []operatorv1.GenericProvider{}

	contract := "v1beta1"

	// Get Core Providers.
	var coreProviderList operatorv1.CoreProviderList

	if err := client.List(ctx, &coreProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of core providers from the server: %w", err)
	}

	if len(coreProviderList.Items) == 1 && coreProviderList.Items[0].Status.Contract != nil {
		contract = *coreProviderList.Items[0].Status.Contract
	}

	for i := range coreProviderList.Items {
		genericProviders = append(genericProviders, &coreProviderList.Items[i])
	}

	// Get Bootstrap Providers.
	var bootstrapProviderList operatorv1.BootstrapProviderList

	if err := client.List(ctx, &bootstrapProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of bootstrap providers from the server: %w", err)
	}

	for i := range bootstrapProviderList.Items {
		genericProviders = append(genericProviders, &bootstrapProviderList.Items[i])
	}

	// Get Control Plane Providers.
	var controlPlaneProviderList operatorv1.ControlPlaneProviderList

	if err := client.List(ctx, &controlPlaneProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of control plane providers from the server: %w", err)
	}

	for i := range controlPlaneProviderList.Items {
		genericProviders = append(genericProviders, &controlPlaneProviderList.Items[i])
	}

	// Get Infrastructure Providers.
	var infrastructureProviderList operatorv1.InfrastructureProviderList

	if err := client.List(ctx, &infrastructureProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of infrastructure providers from the server: %w", err)
	}

	for i := range infrastructureProviderList.Items {
		genericProviders = append(genericProviders, &infrastructureProviderList.Items[i])
	}

	// Get Addon Providers.
	var addonProviderList operatorv1.AddonProviderList

	if err := client.List(ctx, &addonProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of addon providers from the server: %w", err)
	}

	for i := range addonProviderList.Items {
		genericProviders = append(genericProviders, &addonProviderList.Items[i])
	}

	// Get IPAM Providers.
	var ipamProviderList operatorv1.IPAMProviderList

	if err := client.List(ctx, &ipamProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of ipam providers from the server: %w", err)
	}

	for i := range ipamProviderList.Items {
		genericProviders = append(genericProviders, &ipamProviderList.Items[i])
	}

	// Get Runtime Extension Providers.
	var runtimeExtensionProviderList operatorv1.RuntimeExtensionProviderList

	if err := client.List(ctx, &runtimeExtensionProviderList); err != nil {
		return nil, "", fmt.Errorf("cannot get a list of runtime extension providers from the server: %w", err)
	}

	for i := range runtimeExtensionProviderList.Items {
		genericProviders = append(genericProviders, &runtimeExtensionProviderList.Items[i])
	}

	return genericProviders, contract, nil
}

func getProviderFetchConfig(ctx context.Context, genericProvider operatorv1.GenericProvider) (providerSource, providerSourceType, error) {
	// Check that fetch url was provider by user.
	spec := genericProvider.GetSpec()
	if spec.FetchConfig != nil && spec.FetchConfig.URL != "" {
		return providerSource(spec.FetchConfig.URL), providerSourceTypeCustomURL, nil
	}

	// Get fetch url from clusterctl configuration.
	// TODO: support custom clusterctl configuration.
	configClient, err := configclient.New(ctx, "")
	if err != nil {
		return "", "", err
	}

	providerConfig, err := configClient.Providers().Get(genericProvider.ProviderName(), util.ClusterctlProviderType(genericProvider))
	if err != nil {
		// TODO: implement support of fetching data from config maps
		// This is a temporary fix for providers installed from config maps
		if strings.Contains(err.Error(), "failed to get configuration") {
			return "", providerSourceTypeConfigMap, nil
		}

		return "", "", err
	}

	return providerSource(providerConfig.URL()), providerSourceTypeBuiltin, nil
}
