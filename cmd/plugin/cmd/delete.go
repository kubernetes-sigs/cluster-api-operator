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
	"time"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2/textlogger"

	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
)

type deleteOptions struct {
	kubeconfig              string
	kubeconfigContext       string
	coreProvider            bool
	bootstrapProviders      []string
	controlPlaneProviders   []string
	infrastructureProviders []string
	ipamProviders           []string
	addonProviders          []string
	// runtimeExtensionProviders []string
	includeNamespace bool
	includeCRDs      bool
	deleteAll        bool
}

var deleteOpts = &deleteOptions{}

var deleteCmd = &cobra.Command{
	Use:     "delete [providers]",
	GroupID: groupManagement,
	Short:   "Delete one or more providers from the management cluster",
	Long: LongDesc(`
		Delete one or more providers from the management cluster.`),

	Example: Examples(`
		# Deletes the AWS provider
		# Please note that this implies the deletion of all provider components except the hosting namespace
		# and the CRDs.
		capioperator delete --infrastructure aws

		# Deletes all the providers
		# Important! As a consequence of this operation, all the corresponding resources managed by
		# Cluster API Providers are orphaned and there might be ongoing costs incurred as a result of this.
		capioperator delete --all

		# Delete the AWS infrastructure provider and Core provider. This will leave behind Bootstrap and ControlPlane
		# providers
		# Important! As a consequence of this operation, all the corresponding resources managed by
		# the AWS infrastructure provider and Cluster API Providers are orphaned and there might be
		# ongoing costs incurred as a result of this.
		capioperator delete --core --infrastructure aws

		# Delete the AWS infrastructure provider and related CRDs. Please note that this forces deletion of
		# all the related objects (e.g. AWSClusters, AWSMachines etc.).
		# Important! As a consequence of this operation, all the corresponding resources managed by
		# the AWS infrastructure provider are orphaned and there might be ongoing costs incurred as a result of this.
		capioperator delete --infrastructure aws --include-crd

		# Delete the AWS infrastructure provider and its hosting Namespace. Please note that this forces deletion of
		# all objects existing in the namespace.
		# Important! As a consequence of this operation, all the corresponding resources managed by
		# Cluster API Providers are orphaned and there might be ongoing costs incurred as a result of this.
		capioperator delete --infrastructure aws --include-namespace

		# Reset the management cluster to its original state
		# Important! As a consequence of this operation all the corresponding resources on target clouds
		# are "orphaned" and thus there may be ongoing costs incurred as a result of this.
		capioperator delete --all --include-crd  --include-namespace`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDelete()
	},
}

func init() {
	deleteCmd.Flags().StringVar(&deleteOpts.kubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file to use for accessing the management cluster. If unspecified, default discovery rules apply.")
	deleteCmd.Flags().StringVar(&deleteOpts.kubeconfigContext, "kubeconfig-context", "",
		"Context to be used within the kubeconfig file. If empty, current context will be used.")

	deleteCmd.Flags().BoolVar(&deleteOpts.includeNamespace, "include-namespace", false,
		"Forces the deletion of the namespace where the providers are hosted (and of all the contained objects)")
	deleteCmd.Flags().BoolVar(&deleteOpts.includeCRDs, "include-crd", false,
		"Forces the deletion of the provider's CRDs (and of all the related objects)")

	deleteCmd.Flags().BoolVar(&deleteOpts.coreProvider, "core", false,
		"Core provider to delete from the management cluster. If not set, core provider is not removed. Cluster cannot have more then 1 core provider in total.")
	deleteCmd.Flags().StringSliceVarP(&deleteOpts.infrastructureProviders, "infrastructure", "i", nil,
		"Infrastructure provider and namespace (e.g. aws:<namespace>) to delete from the management cluster")
	deleteCmd.Flags().StringSliceVarP(&deleteOpts.bootstrapProviders, "bootstrap", "b", nil,
		"Bootstrap provider and namespace (e.g. kubeadm:<namespace>) to delete from the management cluster")
	deleteCmd.Flags().StringSliceVarP(&deleteOpts.controlPlaneProviders, "control-plane", "c", nil,
		"ControlPlane provider and namespace (e.g. kubeadm:<namespace>) to delete from the management cluster")
	deleteCmd.Flags().StringSliceVar(&deleteOpts.ipamProviders, "ipam", nil,
		"IPAM provider and namespace (e.g. infoblox:<namespace>) to delete from the management cluster")
	// deleteCmd.Flags().StringSliceVar(&deleteOpts.runtimeExtensionProviders, "runtime-extension", nil,
	//	"Runtime extension providers and versions (e.g. test:v0.0.1) to delete from the management cluster")
	deleteCmd.Flags().StringSliceVar(&deleteOpts.addonProviders, "addon", nil,
		"Add-on providers and versions (e.g. helm:<namespace>) to delete from the management cluster")

	deleteCmd.Flags().BoolVar(&deleteOpts.deleteAll, "all", false,
		"Force deletion of all the providers")

	RootCmd.AddCommand(deleteCmd)
}

func runDelete() error {
	ctx := context.Background()

	loggerConfig := textlogger.NewConfig([]textlogger.ConfigOption{}...)
	ctrl.SetLogger(textlogger.NewLogger(loggerConfig))

	hasProviderNames := deleteOpts.coreProvider ||
		(len(deleteOpts.bootstrapProviders) > 0) ||
		(len(deleteOpts.controlPlaneProviders) > 0) ||
		(len(deleteOpts.infrastructureProviders) > 0) ||
		(len(deleteOpts.ipamProviders) > 0) ||
		// (len(deleteOpts.runtimeExtensionProviders) > 0) ||
		(len(deleteOpts.addonProviders) > 0)

	if deleteOpts.deleteAll && hasProviderNames {
		return errors.New("The --all flag can't be used in combination with --core, --bootstrap, --control-plane, --infrastructure, --ipam, --extension, --addon")
	}

	if !deleteOpts.deleteAll && !hasProviderNames {
		return errors.New("At least one of --core, --bootstrap, --control-plane, --infrastructure, --ipam, --extension, --addon should be specified or the --all flag should be set")
	}

	if deleteOpts.kubeconfig == "" {
		deleteOpts.kubeconfig = GetKubeconfigLocation()
	}

	cl, err := CreateKubeClient(deleteOpts.kubeconfig, deleteOpts.kubeconfigContext)
	if err != nil {
		return fmt.Errorf("unable to create client from kubeconfig flag %s with context %s: %w", deleteOpts.kubeconfig, deleteOpts.kubeconfigContext, err)
	}

	group := &DeleteGroup{
		selectors: []fields.Set{},
		providers: []generic.ProviderList{},
	}
	errors := append([]error{},
		group.delete(&operatorv1.BootstrapProviderList{}, deleteOpts.bootstrapProviders...),
		group.delete(&operatorv1.ControlPlaneProviderList{}, deleteOpts.controlPlaneProviders...),
		group.delete(&operatorv1.InfrastructureProviderList{}, deleteOpts.infrastructureProviders...),
		group.delete(&operatorv1.IPAMProviderList{}, deleteOpts.ipamProviders...),
		group.delete(&operatorv1.AddonProviderList{}, deleteOpts.addonProviders...))

	if deleteOpts.coreProvider {
		errors = append(errors, group.delete(&operatorv1.CoreProviderList{}, []string{""}...))
	}

	if err := kerrors.NewAggregate(errors); err != nil {
		return err
	}

	if deleteOpts.deleteAll {
		group.deleteAll()
	}

	return group.execute(ctx, cl)
}

type DeleteGroup struct {
	selectors []fields.Set
	providers []generic.ProviderList
}

func (d *DeleteGroup) delete(providerType generic.ProviderList, names ...string) error {
	for _, provider := range names {
		selector, err := selectorFromProvider(provider)
		if err != nil {
			return fmt.Errorf("invalid provider format: %w", err)
		}

		d.providers = append(d.providers, providerType)
		d.selectors = append(d.selectors, selector)
	}

	return nil
}

func (d *DeleteGroup) deleteAll() {
	for _, list := range operatorv1.ProviderLists {
		providerList, ok := list.(generic.ProviderList)
		if !ok {
			log.V(5).Info("Expected to get GenericProviderList")
			continue
		}

		d.providers = append(d.providers, providerList)
		d.selectors = append(d.selectors, fields.Set{})
	}
}

func (d *DeleteGroup) execute(ctx context.Context, cl ctrlclient.Client) error {
	opts := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Steps:    10,
		Jitter:   0.4,
	}

	log.Info("Waiting for CAPI Operator manifests to be removed...")

	if err := wait.ExponentialBackoff(opts, func() (bool, error) {
		ready := true
		for i := range d.providers {
			if done, err := deleteProviders(ctx, cl, d.providers[i], ctrlclient.MatchingFieldsSelector{
				Selector: fields.SelectorFromSet(d.selectors[i]),
			}); err != nil {
				return false, err
			} else {
				ready = ready && done
			}
		}

		return ready, nil
	}); err != nil {
		return fmt.Errorf("cannot remove provider: %w", err)
	}

	return nil
}

func selectorFromProvider(provider string) (fields.Set, error) {
	var name, namespace string

	parts := strings.Split(provider, ":")
	switch len(parts) {
	case 0 | 3:
	case 1:
		name = parts[0]
	case 2:
		name, namespace = parts[0], parts[1]
	default:
		return nil, fmt.Errorf("invalid provider format: %s", provider)
	}

	selector := fields.Set{}

	if name != "" {
		selector["metadata.name"] = name
	}

	if namespace != "" {
		selector["metadata.namespace"] = namespace
	}

	return selector, nil
}

func deleteProviders(ctx context.Context, client ctrlclient.Client, providerList generic.ProviderList, selector ctrlclient.MatchingFieldsSelector) (bool, error) {
	//nolint:forcetypeassert
	providerList = providerList.DeepCopyObject().(generic.ProviderList)
	ready := true

	gvks, _, err := scheme.ObjectKinds(providerList)
	if err != nil {
		log.Error(err, "Kind is not registered in provider list")
		return false, err
	}

	gvk := gvks[0]

	if err := client.List(ctx, providerList, selector); meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		log.Error(err, fmt.Sprintf("Unable to list providers to delete, %#v", err))
		return false, err
	}

	for _, provider := range providerList.GetItems() {
		log.Info(fmt.Sprintf("Deleting %s %s/%s", provider.GetType(), provider.GetName(), provider.GetNamespace()))

		if err := client.DeleteAllOf(ctx, provider, ctrlclient.InNamespace(provider.GetNamespace())); err != nil {
			return false, fmt.Errorf("unable to issue delete for %s: %w", gvk, err)
		}

		if deleteOpts.includeNamespace {
			if strings.HasPrefix(provider.GetNamespace(), "kube-") || provider.GetNamespace() == "default" {
				log.Info(fmt.Sprintf("Skipping system namespace %s", provider.GetNamespace()))
				continue
			}

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: provider.GetNamespace()}}
			if err := client.Delete(ctx, ns); ctrlclient.IgnoreNotFound(err) != nil {
				return false, fmt.Errorf("unable to issue delete for Namespace %s: %w", provider.GetNamespace(), err)
			}
		}
	}

	if len(providerList.GetItems()) > 0 {
		log.Info(fmt.Sprintf("%d items remaning...", len(providerList.GetItems())))
		return false, nil
	}

	if deleteOpts.includeCRDs && len(providerList.GetItems()) == 0 {
		log.Info("Removing CRDs")

		group := gvk.GroupKind()
		group.Kind = strings.Replace(strings.ToLower(group.Kind), "list", "s", 1)
		crd := &apiextensionsv1.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: group.String()}}

		if err := client.Delete(ctx, crd); ctrlclient.IgnoreNotFound(err) != nil {
			return false, fmt.Errorf("unable to issue delete for %s: %w", group, err)
		}
	}

	log.Info("All requested providers are deleted")

	return ready, nil
}
