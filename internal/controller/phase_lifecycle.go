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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Upgrade ensure all the clusterctl CRDs are available before installing the provider,
// and update existing components if required.
func (p *PhaseReconciler) Upgrade(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Nothing to do if it's a fresh installation.
	if p.provider.GetStatus().InstalledVersion == nil {
		log.V(2).Info("Skipping upgrade, fresh installation detected")

		return &Result{}, nil
	}

	// Provider needs to be re-installed
	if *p.provider.GetStatus().InstalledVersion == p.provider.GetSpec().Version {
		log.V(2).Info("Skipping upgrade, versions match", "version", p.provider.GetSpec().Version)

		return &Result{}, nil
	}

	log.Info("Version changes detected, updating existing components", "installedVersion", *p.provider.GetStatus().InstalledVersion, "targetVersion", p.provider.GetSpec().Version)

	provider := p.providerConverter(p.provider)
	if provider.Version == "" {
		provider.Version = p.options.Version
	}

	if err := p.newClusterClient().ProviderUpgrader().ApplyCustomPlan(ctx, cluster.UpgradeOptions{}, cluster.UpgradeItem{
		NextVersion: p.provider.GetSpec().Version,
		Provider:    provider,
	}); err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsUpgradeErrorReason, operatorv1.ProviderUpgradedCondition)
	}

	log.Info("Provider successfully upgraded", "version", p.provider.GetSpec().Version)
	conditions.Set(p.provider, metav1.Condition{
		Type:    operatorv1.ProviderUpgradedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "ProviderUpgraded",
		Message: "Provider upgraded successfully",
	})

	return &Result{}, nil
}

// Install installs the provider components using clusterctl library.
func (p *PhaseReconciler) Install(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Provider was upgraded, nothing to do
	if p.provider.GetStatus().InstalledVersion != nil && *p.provider.GetStatus().InstalledVersion != p.provider.GetSpec().Version {
		log.V(2).Info("Skipping install, provider was upgraded in this reconciliation")

		return &Result{}, nil
	}

	clusterClient := p.newClusterClient()

	log.Info("Installing provider", "version", p.provider.GetSpec().Version)

	if err := clusterClient.ProviderComponents().Create(ctx, p.components.Objs()); err != nil {
		reason := "InstallFailed"
		if wait.Interrupted(err) {
			reason = "TimedOutWaitingForDeployment"
		}

		return &Result{}, wrapPhaseError(err, reason, operatorv1.ProviderInstalledCondition)
	}

	log.Info("Provider successfully installed", "version", p.provider.GetSpec().Version)
	conditions.Set(p.provider, metav1.Condition{
		Type:    operatorv1.ProviderInstalledCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "ProviderInstalled",
		Message: "Provider installed successfully",
	})

	return &Result{}, nil
}

func convertProvider(provider operatorv1.GenericProvider) clusterctlv1.Provider {
	clusterctlProvider := &clusterctlv1.Provider{}
	clusterctlProvider.Name = clusterctlProviderName(provider).Name
	clusterctlProvider.Namespace = provider.GetNamespace()
	clusterctlProvider.Type = string(util.ClusterctlProviderType(provider))
	clusterctlProvider.ProviderName = provider.ProviderName()

	if provider.GetStatus().InstalledVersion != nil {
		clusterctlProvider.Version = *provider.GetStatus().InstalledVersion
	}

	return *clusterctlProvider
}

// Delete deletes the provider components using clusterctl library.
func (p *PhaseReconciler) Delete(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting provider", "version", p.provider.GetSpec().Version)

	clusterClient := p.newClusterClient()

	provider := p.providerConverter(p.provider)
	if provider.Version == "" {
		provider.Version = p.options.Version
	}

	err := clusterClient.ProviderComponents().Delete(ctx, cluster.DeleteOptions{
		Provider:         provider,
		IncludeNamespace: false,
		IncludeCRDs:      false,
	})
	if err == nil {
		log.Info("Provider successfully deleted", "version", p.provider.GetSpec().Version)
	}

	return &Result{}, wrapPhaseError(err, operatorv1.OldComponentsDeletionErrorReason, operatorv1.ProviderInstalledCondition)
}

func clusterctlProviderName(provider operatorv1.GenericProvider) client.ObjectKey {
	prefix := ""

	switch provider.(type) {
	case *operatorv1.BootstrapProvider:
		prefix = "bootstrap-"
	case *operatorv1.ControlPlaneProvider:
		prefix = "control-plane-"
	case *operatorv1.InfrastructureProvider:
		prefix = "infrastructure-"
	case *operatorv1.AddonProvider:
		prefix = "addon-"
	case *operatorv1.IPAMProvider:
		prefix = "ipam-"
	case *operatorv1.RuntimeExtensionProvider:
		prefix = "runtime-extension-"
	}

	return client.ObjectKey{Name: prefix + provider.ProviderName(), Namespace: provider.GetNamespace()}
}

func (p *PhaseReconciler) repositoryProxy(ctx context.Context, provider configclient.Provider, configClient configclient.Client, options ...repository.Option) (repository.Client, error) {
	injectRepo := p.repo

	if !provider.SameAs(p.providerConfig) {
		genericProvider, err := p.providerMapper(ctx, provider)
		if err != nil {
			return nil, wrapPhaseError(err, "unable to find generic provider for configclient "+string(provider.Type())+": "+provider.Name(), operatorv1.ProviderUpgradedCondition)
		}

		if exists, err := p.checkConfigMapExists(ctx, *providerLabelSelector(genericProvider), genericProvider.GetNamespace()); err != nil {
			provider := client.ObjectKeyFromObject(genericProvider)
			return nil, wrapPhaseError(err, "failed to check the config map repository existence for provider "+provider.String(), operatorv1.ProviderUpgradedCondition)
		} else if !exists {
			provider := client.ObjectKeyFromObject(genericProvider)
			return nil, wrapPhaseError(fmt.Errorf("config map not found"), "config map repository required for validation does not exist yet for provider "+provider.String(), operatorv1.ProviderUpgradedCondition)
		}

		repo, err := p.configmapRepository(ctx, providerLabelSelector(genericProvider), InNamespace(genericProvider.GetNamespace()), SkipComponents{})
		if err != nil {
			provider := client.ObjectKeyFromObject(genericProvider)
			return nil, wrapPhaseError(err, "failed to load the repository for provider "+provider.String(), operatorv1.ProviderUpgradedCondition)
		}

		injectRepo = repo
	}

	cl, err := repository.New(ctx, provider, configClient, append([]repository.Option{repository.InjectRepository(injectRepo)}, options...)...)
	if err != nil {
		return nil, err
	}

	return repositoryProxy{Client: cl, components: p.components}, nil
}

// newClusterClient returns a clusterctl client for interacting with management cluster.
func (p *PhaseReconciler) newClusterClient() cluster.Client {
	return cluster.New(cluster.Kubeconfig{}, p.configClient, cluster.InjectProxy(&controllerProxy{
		ctrlClient: clientProxy{p.ctrlClient, p.providerLister},
		ctrlConfig: p.ctrlConfig,
	}), cluster.InjectRepositoryFactory(p.repositoryProxy))
}
