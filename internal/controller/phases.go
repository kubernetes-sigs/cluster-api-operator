/*
Copyright 2022 The Kubernetes Authors.

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
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/yamlprocessor"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	metadataFile = "metadata.yaml"
)

// phaseReconciler holds all required information for interacting with clusterctl code and
// helps to iterate through provider reconciliation phases.
type phaseReconciler struct {
	provider     genericprovider.GenericProvider
	providerList genericprovider.GenericProviderList

	ctrlClient         client.Client
	ctrlConfig         *rest.Config
	repo               repository.Repository
	contract           string
	options            repository.ComponentsOptions
	providerConfig     configclient.Provider
	configClient       configclient.Client
	overridesClient    configclient.Client
	components         repository.Components
	clusterctlProvider *clusterctlv1.Provider
}

// reconcilePhaseFn is a function that represent a phase of the reconciliation.
type reconcilePhaseFn func(context.Context) (reconcile.Result, error)

// PhaseError custom error type for phases.
type PhaseError struct {
	Reason   string
	Type     clusterv1.ConditionType
	Severity clusterv1.ConditionSeverity
	Err      error
}

func (p *PhaseError) Error() string {
	return p.Err.Error()
}

func wrapPhaseError(err error, reason string, condition clusterv1.ConditionType) error {
	if err == nil {
		return nil
	}

	return &PhaseError{
		Err:      err,
		Type:     condition,
		Reason:   reason,
		Severity: clusterv1.ConditionSeverityWarning,
	}
}

// newPhaseReconciler returns phase reconciler for the given provider.
func newPhaseReconciler(r GenericProviderReconciler, provider genericprovider.GenericProvider, providerList genericprovider.GenericProviderList) *phaseReconciler {
	return &phaseReconciler{
		ctrlClient:         r.Client,
		ctrlConfig:         r.Config,
		clusterctlProvider: &clusterctlv1.Provider{},
		provider:           provider,
		providerList:       providerList,
	}
}

// preflightChecks a wrapper around the preflight checks.
func (p *phaseReconciler) preflightChecks(ctx context.Context) (reconcile.Result, error) {
	return preflightChecks(ctx, p.ctrlClient, p.provider, p.providerList)
}

// initializePhaseReconciler initializes phase reconciler.
func (p *phaseReconciler) initializePhaseReconciler(ctx context.Context) (reconcile.Result, error) {
	path := configPath
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		path = ""
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Initialize a client for interacting with the clusterctl configuration.
	initConfig, err := configclient.New(ctx, path)
	if err != nil {
		return reconcile.Result{}, err
	} else if path != "" {
		// Set the image and providers override client
		p.overridesClient = initConfig
	}

	overrideProviders := []configclient.Provider{}

	if p.overridesClient != nil {
		providers, err := p.overridesClient.Providers().List()
		if err != nil {
			return reconcile.Result{}, err
		}

		overrideProviders = providers
	}

	reader, err := p.secretReader(ctx, overrideProviders...)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Load provider's secret and config url.
	p.configClient, err = configclient.New(ctx, "", configclient.InjectReader(reader))
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load the secret reader", operatorv1.ProviderInstalledCondition)
	}

	// Get returns the configuration for the provider with a given name/type.
	// This is done using clusterctl internal API types.
	p.providerConfig, err = p.configClient.Providers().Get(p.provider.GetName(), util.ClusterctlProviderType(p.provider))
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.UnknownProviderReason, operatorv1.ProviderInstalledCondition)
	}

	return reconcile.Result{}, nil
}

// load provider specific configuration into phaseReconciler object.
func (p *phaseReconciler) load(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Loading provider")

	var err error

	spec := p.provider.GetSpec()

	labelSelector := &metav1.LabelSelector{
		MatchLabels: p.prepareConfigMapLabels(),
	}

	// Replace label selector if user wants to use custom config map
	if p.provider.GetSpec().FetchConfig != nil && p.provider.GetSpec().FetchConfig.Selector != nil {
		labelSelector = p.provider.GetSpec().FetchConfig.Selector
	}

	additionalManifests, err := p.fetchAddionalManifests(ctx)
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load additional manifests", operatorv1.ProviderInstalledCondition)
	}

	p.repo, err = p.configmapRepository(ctx, labelSelector, p.provider.GetNamespace(), additionalManifests)
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load the repository", operatorv1.ProviderInstalledCondition)
	}

	if spec.Version == "" {
		// User didn't set the version, so we need to find the latest one from the matching config maps.
		repoVersions, err := p.repo.GetVersions(ctx)
		if err != nil {
			return reconcile.Result{}, wrapPhaseError(err, fmt.Sprintf("failed to get a list of available versions for provider %q", p.provider.GetName()), operatorv1.ProviderInstalledCondition)
		}

		spec.Version, err = getLatestVersion(repoVersions)
		if err != nil {
			return reconcile.Result{}, wrapPhaseError(err, fmt.Sprintf("failed to get the latest version for provider %q", p.provider.GetName()), operatorv1.ProviderInstalledCondition)
		}

		// Add latest version to the provider spec.
		p.provider.SetSpec(spec)
	}

	// Store some provider specific inputs for passing it to clusterctl library
	p.options = repository.ComponentsOptions{
		TargetNamespace:     p.provider.GetNamespace(),
		SkipTemplateProcess: false,
		Version:             spec.Version,
	}

	if err := p.validateRepoCAPIVersion(ctx); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.CAPIVersionIncompatibilityReason, operatorv1.ProviderInstalledCondition)
	}

	return reconcile.Result{}, nil
}

// secretReader use clusterctl MemoryReader structure to store the configuration variables
// that are obtained from a secret and try to set fetch url config.
func (p *phaseReconciler) secretReader(ctx context.Context, providers ...configclient.Provider) (configclient.Reader, error) {
	log := ctrl.LoggerFrom(ctx)

	mr := configclient.NewMemoryReader()

	if err := mr.Init(ctx, ""); err != nil {
		return nil, err
	}

	// Fetch configuration variables from the secret. See API field docs for more info.
	if p.provider.GetSpec().ConfigSecret != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: p.provider.GetSpec().ConfigSecret.Namespace, Name: p.provider.GetSpec().ConfigSecret.Name}

		if err := p.ctrlClient.Get(ctx, key, secret); err != nil {
			return nil, err
		}

		for k, v := range secret.Data {
			mr.Set(k, string(v))
		}
	} else {
		log.Info("No configuration secret was specified")
	}

	for _, provider := range providers {
		if _, err := mr.AddProvider(provider.Name(), provider.Type(), provider.URL()); err != nil {
			return nil, err
		}
	}

	// If provided store fetch config url in memory reader.
	if p.provider.GetSpec().FetchConfig != nil {
		if p.provider.GetSpec().FetchConfig.URL != "" {
			log.Info("Custom fetch configuration url was provided")
			return mr.AddProvider(p.provider.GetName(), util.ClusterctlProviderType(p.provider), p.provider.GetSpec().FetchConfig.URL)
		}

		if p.provider.GetSpec().FetchConfig.Selector != nil {
			log.Info("Custom fetch configuration config map was provided")

			// To register a new provider from the config map, we need to specify a URL with a valid
			// format. However, since we're using data from a local config map, URLs are not needed.
			// As a workaround, we add a fake but well-formatted URL.

			fakeURL := "https://example.com/my-provider"

			return mr.AddProvider(p.provider.GetName(), util.ClusterctlProviderType(p.provider), fakeURL)
		}
	}

	return mr, nil
}

// configmapRepository use clusterctl NewMemoryRepository structure to store the manifests
// and metadata from a given configmap.
func (p *phaseReconciler) configmapRepository(ctx context.Context, labelSelector *metav1.LabelSelector, namespace, additionalManifests string) (repository.Repository, error) {
	mr := repository.NewMemoryRepository()
	mr.WithPaths("", "components.yaml")

	cml := &corev1.ConfigMapList{}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	if err = p.ctrlClient.List(ctx, cml, &client.ListOptions{LabelSelector: selector, Namespace: namespace}); err != nil {
		return nil, err
	}

	if len(cml.Items) == 0 {
		return nil, fmt.Errorf("no ConfigMaps found with selector %s", labelSelector.String())
	}

	for _, cm := range cml.Items {
		version := cm.Name
		errMsg := "from the Name"

		if cm.Labels != nil {
			ver, ok := cm.Labels[operatorv1.ConfigMapVersionLabelName]
			if ok {
				version = ver
				errMsg = "from the Label " + operatorv1.ConfigMapVersionLabelName
			}
		}

		if _, err = versionutil.ParseSemantic(version); err != nil {
			return nil, fmt.Errorf("ConfigMap %s/%s has invalid version:%s (%s)", cm.Namespace, cm.Name, version, errMsg)
		}

		metadata, ok := cm.Data[metadataConfigMapKey]
		if !ok {
			return nil, fmt.Errorf("ConfigMap %s/%s has no metadata", cm.Namespace, cm.Name)
		}

		mr.WithFile(version, metadataFile, []byte(metadata))

		components, err := getComponentsData(cm)
		if err != nil {
			return nil, err
		}

		if additionalManifests != "" {
			components = components + "\n---\n" + additionalManifests
		}

		mr.WithFile(version, mr.ComponentsPath(), []byte(components))
	}

	return mr, nil
}

func (p *phaseReconciler) fetchAddionalManifests(ctx context.Context) (string, error) {
	cm := &corev1.ConfigMap{}

	if p.provider.GetSpec().AdditionalManifestsRef != nil {
		key := types.NamespacedName{Namespace: p.provider.GetSpec().AdditionalManifestsRef.Namespace, Name: p.provider.GetSpec().AdditionalManifestsRef.Name}

		if err := p.ctrlClient.Get(ctx, key, cm); err != nil {
			return "", fmt.Errorf("failed to get ConfigMap %s/%s: %w", key.Namespace, key.Name, err)
		}
	}

	return cm.Data[additionalManifestsConfigMapKey], nil
}

// getComponentsData returns components data based on if it's compressed or not.
func getComponentsData(cm corev1.ConfigMap) (string, error) {
	// Data is not compressed, return it immediately.
	if cm.GetAnnotations()[compressedAnnotation] != "true" {
		components, ok := cm.Data[componentsConfigMapKey]
		if !ok {
			return "", fmt.Errorf("ConfigMap %s/%s Data has no components", cm.Namespace, cm.Name)
		}

		return components, nil
	}

	// Otherwise we have to decompress the data first.
	compressedComponents, ok := cm.BinaryData[componentsConfigMapKey]
	if !ok {
		return "", fmt.Errorf("ConfigMap %s/%s BinaryData has no components", cm.Namespace, cm.Name)
	}

	zr, err := gzip.NewReader(bytes.NewReader(compressedComponents))
	if err != nil {
		return "", err
	}

	components, err := io.ReadAll(zr)
	if err != nil {
		return "", fmt.Errorf("cannot decompress data from ConfigMap %s/%s", cm.Namespace, cm.Name)
	}

	if err := zr.Close(); err != nil {
		return "", err
	}

	return string(components), nil
}

// validateRepoCAPIVersion checks that the repo is using the correct version.
func (p *phaseReconciler) validateRepoCAPIVersion(ctx context.Context) error {
	name := p.provider.GetName()

	file, err := p.repo.GetFile(ctx, p.options.Version, metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read %q from the repository for provider %q: %w", metadataFile, name, err)
	}

	// Convert the yaml into a typed object
	latestMetadata := &clusterctlv1.Metadata{}
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)

	if err := runtime.DecodeInto(codecFactory.UniversalDecoder(), file, latestMetadata); err != nil {
		return fmt.Errorf("error decoding %q for provider %q: %w", metadataFile, name, err)
	}

	// Gets the contract for the target release.
	targetVersion, err := versionutil.ParseSemantic(p.options.Version)
	if err != nil {
		return fmt.Errorf("failed to parse current version for the %s provider: %w", name, err)
	}

	releaseSeries := latestMetadata.GetReleaseSeriesForVersion(targetVersion)
	if releaseSeries == nil {
		return fmt.Errorf("invalid provider metadata: version %s for the provider %s does not match any release series", p.options.Version, name)
	}

	if releaseSeries.Contract != "v1alpha4" && releaseSeries.Contract != "v1beta1" {
		return fmt.Errorf(capiVersionIncompatibilityMessage, clusterv1.GroupVersion.Version, releaseSeries.Contract, name)
	}

	p.contract = releaseSeries.Contract

	return nil
}

// fetch fetches the provider components from the repository and processes all yaml manifests.
func (p *phaseReconciler) fetch(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching provider")

	// Fetch the provider components yaml file from the provided repository GitHub/GitLab/ConfigMap.
	componentsFile, err := p.repo.GetFile(ctx, p.options.Version, p.repo.ComponentsPath())
	if err != nil {
		err = fmt.Errorf("failed to read %q from provider's repository %q: %w", p.repo.ComponentsPath(), p.providerConfig.ManifestLabel(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Generate a set of new objects using the clusterctl library. NewComponents() will do the yaml processing,
	// like ensure all the provider components are in proper namespace, replace variables, etc. See the clusterctl
	// documentation for more details.
	p.components, err = repository.NewComponents(repository.ComponentsInput{
		Provider:     p.providerConfig,
		ConfigClient: p.configClient,
		Processor:    yamlprocessor.NewSimpleProcessor(),
		RawYaml:      componentsFile,
		Options:      p.options,
	})
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// ProviderSpec provides fields for customizing the provider deployment options.
	// We can use clusterctl library to apply this customizations.
	if err := repository.AlterComponents(p.components, customizeObjectsFn(p.provider)); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Apply patches to the provider components if specified.
	if err := repository.AlterComponents(p.components, applyPatches(ctx, p.provider)); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Apply image overrides to the provider manifests.
	if err := repository.AlterComponents(p.components, imageOverrides(p.components.ManifestLabel(), p.overridesClient)); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	conditions.Set(p.provider, conditions.TrueCondition(operatorv1.ProviderInstalledCondition))

	return reconcile.Result{}, nil
}

// upgrade ensure all the clusterctl CRDs are available before installing the provider,
// and update existing components if required.
func (p *phaseReconciler) upgrade(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Nothing to do if it's a fresh installation.
	if p.provider.GetStatus().InstalledVersion == nil {
		return reconcile.Result{}, nil
	}

	// Provider needs to be re-installed
	if *p.provider.GetStatus().InstalledVersion == p.provider.GetSpec().Version {
		return reconcile.Result{}, nil
	}

	log.Info("Version changes detected, updating existing components")

	if err := p.newClusterClient().ProviderUpgrader().ApplyCustomPlan(ctx, cluster.UpgradeOptions{}, cluster.UpgradeItem{
		NextVersion: p.provider.GetSpec().Version,
		Provider:    getProvider(p.provider, p.options.Version),
	}); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsUpgradeErrorReason, operatorv1.ProviderUpgradedCondition)
	}

	log.Info("Provider successfully upgraded")
	conditions.Set(p.provider, conditions.TrueCondition(operatorv1.ProviderUpgradedCondition))

	return reconcile.Result{}, nil
}

// install installs the provider components using clusterctl library.
func (p *phaseReconciler) install(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Provider was upgraded, nothing to do
	if p.provider.GetStatus().InstalledVersion != nil && *p.provider.GetStatus().InstalledVersion != p.provider.GetSpec().Version {
		return reconcile.Result{}, nil
	}

	clusterClient := p.newClusterClient()

	log.Info("Installing provider")

	if err := clusterClient.ProviderComponents().Create(ctx, p.components.Objs()); err != nil {
		reason := "Install failed"
		if wait.Interrupted(err) {
			reason = "Timed out waiting for deployment to become ready"
		}

		return reconcile.Result{}, wrapPhaseError(err, reason, operatorv1.ProviderInstalledCondition)
	}

	log.Info("Provider successfully installed")
	conditions.Set(p.provider, conditions.TrueCondition(operatorv1.ProviderInstalledCondition))

	return reconcile.Result{}, nil
}

func (p *phaseReconciler) reportStatus(ctx context.Context) (reconcile.Result, error) {
	status := p.provider.GetStatus()
	status.Contract = &p.contract
	installedVersion := p.components.Version()
	status.InstalledVersion = &installedVersion
	p.provider.SetStatus(status)

	return reconcile.Result{}, nil
}

func getProvider(provider operatorv1.GenericProvider, defaultVersion string) clusterctlv1.Provider {
	clusterctlProvider := &clusterctlv1.Provider{}
	clusterctlProvider.Name = clusterctlProviderName(provider).Name
	clusterctlProvider.Namespace = provider.GetNamespace()
	clusterctlProvider.Type = string(util.ClusterctlProviderType(provider))
	clusterctlProvider.ProviderName = provider.GetName()

	if provider.GetStatus().InstalledVersion != nil {
		clusterctlProvider.Version = *provider.GetStatus().InstalledVersion
	} else {
		clusterctlProvider.Version = defaultVersion
	}

	return *clusterctlProvider
}

// delete deletes the provider components using clusterctl library.
func (p *phaseReconciler) delete(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting provider")

	clusterClient := p.newClusterClient()

	err := clusterClient.ProviderComponents().Delete(ctx, cluster.DeleteOptions{
		Provider:         getProvider(p.provider, p.options.Version),
		IncludeNamespace: false,
		IncludeCRDs:      false,
	})

	return reconcile.Result{}, wrapPhaseError(err, operatorv1.OldComponentsDeletionErrorReason, operatorv1.ProviderInstalledCondition)
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

	return client.ObjectKey{Name: prefix + provider.GetName(), Namespace: provider.GetNamespace()}
}

func (p *phaseReconciler) repositoryProxy(ctx context.Context, provider configclient.Provider, configClient configclient.Client, options ...repository.Option) (repository.Client, error) {
	injectRepo := p.repo

	if !provider.SameAs(p.providerConfig) {
		genericProvider, err := util.GetGenericProvider(ctx, p.ctrlClient, provider)
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

		repo, err := p.configmapRepository(ctx, providerLabelSelector(genericProvider), genericProvider.GetNamespace(), "")
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
func (p *phaseReconciler) newClusterClient() cluster.Client {
	return cluster.New(cluster.Kubeconfig{}, p.configClient, cluster.InjectProxy(&controllerProxy{
		ctrlClient: clientProxy{p.ctrlClient},
		ctrlConfig: p.ctrlConfig,
	}), cluster.InjectRepositoryFactory(p.repositoryProxy))
}

func getLatestVersion(repoVersions []string) (string, error) {
	if len(repoVersions) == 0 {
		err := fmt.Errorf("no versions available")

		return "", wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Initialize latest version with the first element value.
	latestVersion := versionutil.MustParseSemantic(repoVersions[0])

	for _, versionString := range repoVersions {
		parsedVersion, err := versionutil.ParseSemantic(versionString)
		if err != nil {
			return "", wrapPhaseError(err, fmt.Sprintf("cannot parse version string: %s", versionString), operatorv1.ProviderInstalledCondition)
		}

		if latestVersion.LessThan(parsedVersion) {
			latestVersion = parsedVersion
		}
	}

	return "v" + latestVersion.String(), nil
}
