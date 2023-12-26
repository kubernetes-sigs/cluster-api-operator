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

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
)

type upgradeApplyOptions struct {
	kubeconfig              string
	kubeconfigContext       string
	contract                string
	coreProvider            string
	bootstrapProviders      []string
	controlPlaneProviders   []string
	infrastructureProviders []string
	ipamProviders           []string
	// runtimeExtensionProviders []string
	addonProviders      []string
	waitProviders       bool
	waitProviderTimeout int
}

var upgradeApplyOpts = &upgradeApplyOptions{}

var upgradeApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply new versions of Cluster API core and providers in a management cluster",
	Long: LongDesc(`
		The upgrade apply command applies new versions of Cluster API providers as defined by capioperator upgrade plan.

		New version should be applied ensuring all the providers uses the same cluster API version
		in order to guarantee the proper functioning of the management cluster.

 		Specifying the provider using namespace/name:version is deprecated and will be dropped in a future release.`),

	Example: Examples(`
		# Upgrades all the providers in the management cluster to the latest version available which is compliant
		# to the v1alpha4 API Version of Cluster API (contract).
		capioperator upgrade apply --contract v1alpha4

		# Upgrades only the aws provider to the v2.0.1 version.
		capioperator upgrade apply --infrastructure aws:v2.0.1`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgradeApply()
	},
}

func init() {
	upgradeApplyCmd.Flags().StringVar(&upgradeApplyOpts.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file to use for accessing the management cluster. If unspecified, default discovery rules apply.")
	upgradeApplyCmd.Flags().StringVar(&upgradeApplyOpts.kubeconfigContext, "kubeconfig-context", "",
		"Context to be used within the kubeconfig file. If empty, current context will be used.")
	upgradeApplyCmd.Flags().StringVar(&upgradeApplyOpts.contract, "contract", "",
		"The API Version of Cluster API (contract, e.g. v1alpha4) the management cluster should upgrade to")

	upgradeApplyCmd.Flags().StringVar(&upgradeApplyOpts.coreProvider, "core", "",
		"Core provider instance version (e.g. cluster-api:v1.1.5) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().StringSliceVarP(&upgradeApplyOpts.infrastructureProviders, "infrastructure", "i", nil,
		"Infrastructure providers instance and versions (e.g. aws:v2.0.1) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().StringSliceVarP(&upgradeApplyOpts.bootstrapProviders, "bootstrap", "b", nil,
		"Bootstrap providers instance and versions (e.g. kubeadm:v1.1.5) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().StringSliceVarP(&upgradeApplyOpts.controlPlaneProviders, "control-plane", "c", nil,
		"ControlPlane providers instance and versions (e.g. kubeadm:v1.1.5) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().StringSliceVar(&upgradeApplyOpts.ipamProviders, "ipam", nil,
		"IPAM providers and versions (e.g. infoblox:v0.0.1) to upgrade to. This flag can be used as alternative to --contract.")
	// upgradeApplyCmd.Flags().StringSliceVar(&upgradeApplyOpts.runtimeExtensionProviders, "runtime-extension", nil,
	//	"Runtime extension providers and versions (e.g. test:v0.0.1) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().StringSliceVar(&upgradeApplyOpts.addonProviders, "addon", nil,
		"Add-on providers and versions (e.g. helm:v0.1.0) to upgrade to. This flag can be used as alternative to --contract.")
	upgradeApplyCmd.Flags().BoolVar(&upgradeApplyOpts.waitProviders, "wait-providers", false,
		"Wait for providers to be upgraded.")
	upgradeApplyCmd.Flags().IntVar(&upgradeApplyOpts.waitProviderTimeout, "wait-provider-timeout", 5*60,
		"Wait timeout per provider upgrade in seconds. This value is ignored if --wait-providers is false")
}

func runUpgradeApply() error {
	ctx := context.Background()

	hasProviderNames := (upgradeApplyOpts.coreProvider != "") ||
		(len(upgradeApplyOpts.bootstrapProviders) > 0) ||
		(len(upgradeApplyOpts.controlPlaneProviders) > 0) ||
		(len(upgradeApplyOpts.infrastructureProviders) > 0) ||
		// (len(upgradeApplyOpts.ipamProviders) > 0) ||
		// (len(upgradeApplyOpts.runtimeExtensionProviders) > 0) ||
		(len(upgradeApplyOpts.addonProviders) > 0)

	if upgradeApplyOpts.contract == "" && !hasProviderNames {
		return errors.New("Either the --contract flag or at least one of the following flags has to be set: --core, --bootstrap, --control-plane, --infrastructure, --ipam, --extension, --addon")
	}

	if upgradeApplyOpts.contract != "" && hasProviderNames {
		return errors.New("The --contract flag can't be used in combination with --core, --bootstrap, --control-plane, --infrastructure, --ipam, --extension, --addon")
	}

	return upgradeProvider(ctx, upgradeApplyOpts)
}

func upgradeProvider(ctx context.Context, opts *upgradeApplyOptions) error {
	return errors.New("Not implemented")
}
