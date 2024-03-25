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

package phases

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
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/internal/controller/proxy"
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

// PhaseReconciler holds all required information for interacting with clusterctl code and
// helps to iterate through provider reconciliation phases.
type PhaseReconciler[P generic.Provider, G generic.Group[P]] struct {
	ctrlClient     client.Client
	Repo           repository.Repository
	Contract       string
	Options        repository.ComponentsOptions
	ProviderConfig configclient.Provider
	ConfigClient   configclient.Client
	Components     repository.Components
}

type Phase[P generic.Provider] struct {
	client.Client
	Provider           P
	Config             *rest.Config
	ProviderList       generic.ProviderList
	ProviderType       clusterctlv1.ProviderType
	ClusterctlProvider *clusterctlv1.Provider
}

var _ generic.Group[generic.Provider] = Phase[generic.Provider]{}

func NewPhase[P generic.Provider](provider P, providerList generic.ProviderList, src generic.GroupBuilder[P]) generic.Group[P] {
	return Phase[P]{
		Provider:           provider,
		ProviderList:       providerList,
		Client:             src.GetClient(),
		Config:             src.GetConfig(),
		ProviderType:       src.ClusterctlProviderType(),
		ClusterctlProvider: src.ClusterctlProvider(provider),
	}
}

// ClusterctlProviderType implements generic.ProviderGroup.
func (p Phase[P]) ClusterctlProviderType() clusterctlv1.ProviderType {
	return p.ProviderType
}

// GenericProvider implements generic.ProviderGroup.
func (p Phase[P]) GenericProvider() generic.Provider {
	return p.Provider
}

// GetClient implements generic.ProviderGroup.
func (p Phase[P]) GetClient() client.Client {
	return p.Client
}

// GetConfig implements generic.ProviderGroup.
func (p Phase[P]) GetConfig() *rest.Config {
	return p.Config
}

// GetProvider implements generic.ProviderGroup.
func (p Phase[P]) GetProvider() P {
	return p.Provider
}

// GetClusterctlProvider implements generic.ProviderGroup.
func (p Phase[P]) GetClusterctlProvider() *clusterctlv1.Provider {
	return p.ClusterctlProvider
}

// GetProviderList implements generic.ProviderGroup.
func (p Phase[P]) GetProviderList() generic.ProviderList {
	return p.ProviderList
}

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

// NewPhaseReconciler returns phase reconciler for the given phase.GetProvider().
func NewPhaseReconciler[P generic.Provider, G generic.Group[P]](cl client.Client) *PhaseReconciler[P, G] {
	return &PhaseReconciler[P, G]{
		ctrlClient: cl,
	}
}

// InitializePhaseReconciler initializes phase reconciler.
func (p *PhaseReconciler[P, G]) InitializePhaseReconciler(ctx context.Context, phase G) (reconcile.Result, error) {
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
	}

	providers, err := initConfig.Providers().List()
	if err != nil {
		return reconcile.Result{}, err
	}

	reader, err := secretReader(ctx, phase, providers...)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Load provider's secret and config url.
	p.ConfigClient, err = configclient.New(ctx, "", configclient.InjectReader(reader))
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load the secret reader", operatorv1.ProviderInstalledCondition)
	}

	// Get returns the configuration for the provider with a given name/type.
	// This is done using clusterctl internal API types.
	p.ProviderConfig, err = p.ConfigClient.Providers().Get(phase.GetProvider().GetName(), phase.ClusterctlProviderType())
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.UnknownProviderReason, operatorv1.ProviderInstalledCondition)
	}

	return reconcile.Result{}, nil
}

// Load provider specific configuration into phaseReconciler object.
func (p *PhaseReconciler[P, G]) Load(ctx context.Context, phase G) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Loading provider")

	var err error

	spec := phase.GetProvider().GetSpec()

	labelSelector := &metav1.LabelSelector{
		MatchLabels: providerLabels(phase.GetProvider()),
	}

	// Replace label selector if user wants to use custom config map
	if phase.GetProvider().GetSpec().FetchConfig != nil && phase.GetProvider().GetSpec().FetchConfig.Selector != nil {
		labelSelector = phase.GetProvider().GetSpec().FetchConfig.Selector
	}

	additionalManifests, err := fetchAddionalManifests(ctx, phase)
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load additional manifests", operatorv1.ProviderInstalledCondition)
	}

	p.Repo, err = configmapRepository(ctx, phase.GetClient(), labelSelector, phase.GetProvider().GetNamespace(), additionalManifests)
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to load the repository", operatorv1.ProviderInstalledCondition)
	}

	if spec.Version == "" {
		// User didn't set the version, so we need to find the latest one from the matching config maps.
		repoVersions, err := p.Repo.GetVersions(ctx)
		if err != nil {
			return reconcile.Result{}, wrapPhaseError(err, fmt.Sprintf("failed to get a list of available versions for provider %q", phase.GetProvider().GetName()), operatorv1.ProviderInstalledCondition)
		}

		spec.Version, err = getLatestVersion(repoVersions)
		if err != nil {
			return reconcile.Result{}, wrapPhaseError(err, fmt.Sprintf("failed to get the latest version for provider %q", phase.GetProvider().GetName()), operatorv1.ProviderInstalledCondition)
		}

		// Add latest version to the provider spec.
		phase.GetProvider().SetSpec(spec)
	}

	// Store some provider specific inputs for passing it to clusterctl library
	p.Options = repository.ComponentsOptions{
		TargetNamespace:     phase.GetProvider().GetNamespace(),
		SkipTemplateProcess: false,
		Version:             spec.Version,
	}

	if err := p.validateRepoCAPIVersion(ctx, phase); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.CAPIVersionIncompatibilityReason, operatorv1.ProviderInstalledCondition)
	}

	return reconcile.Result{}, nil
}

// secretReader use clusterctl MemoryReader structure to store the configuration variables
// that are obtained from a secret and try to set fetch url config.
func secretReader[P generic.Provider](ctx context.Context, phase generic.Group[P], providers ...configclient.Provider) (configclient.Reader, error) {
	log := ctrl.LoggerFrom(ctx)

	mr := configclient.NewMemoryReader()

	if err := mr.Init(ctx, ""); err != nil {
		return nil, err
	}

	// Fetch configuration variables from the secret. See API field docs for more info.
	if phase.GetProvider().GetSpec().ConfigSecret != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: phase.GetProvider().GetSpec().ConfigSecret.Namespace, Name: phase.GetProvider().GetSpec().ConfigSecret.Name}

		if err := phase.GetClient().Get(ctx, key, secret); err != nil {
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
	if phase.GetProvider().GetSpec().FetchConfig != nil {
		if phase.GetProvider().GetSpec().FetchConfig.URL != "" {
			log.Info("Custom fetch configuration url was provided")
			return mr.AddProvider(phase.GetProvider().GetName(), phase.ClusterctlProviderType(), phase.GetProvider().GetSpec().FetchConfig.URL)
		}

		if phase.GetProvider().GetSpec().FetchConfig.Selector != nil {
			log.Info("Custom fetch configuration config map was provided")

			// To register a new provider from the config map, we need to specify a URL with a valid
			// format. However, since we're using data from a local config map, URLs are not needed.
			// As a workaround, we add a fake but well-formatted URL.

			fakeURL := "https://example.com/my-provider"

			return mr.AddProvider(phase.GetProvider().GetName(), phase.ClusterctlProviderType(), fakeURL)
		}
	}

	return mr, nil
}

// configmapRepository use clusterctl NewMemoryRepository structure to store the manifests
// and metadata from a given configmap.
func configmapRepository(ctx context.Context, cl client.Client, labelSelector *metav1.LabelSelector, namespace, additionalManifests string) (repository.Repository, error) {
	mr := repository.NewMemoryRepository()
	mr.WithPaths("", "components.yaml")

	cml := &corev1.ConfigMapList{}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	if err = cl.List(ctx, cml, &client.ListOptions{LabelSelector: selector, Namespace: namespace}); err != nil {
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

func fetchAddionalManifests[P generic.Provider, G generic.Group[P]](ctx context.Context, phase G) (string, error) {
	cm := &corev1.ConfigMap{}

	if phase.GetProvider().GetSpec().AdditionalManifestsRef != nil {
		key := types.NamespacedName{Namespace: phase.GetProvider().GetSpec().AdditionalManifestsRef.Namespace, Name: phase.GetProvider().GetSpec().AdditionalManifestsRef.Name}

		if err := phase.GetClient().Get(ctx, key, cm); err != nil {
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
func (p *PhaseReconciler[P, G]) validateRepoCAPIVersion(ctx context.Context, phase G) error {
	name := phase.GetProvider().GetName()

	file, err := p.Repo.GetFile(ctx, p.Options.Version, metadataFile)
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
	targetVersion, err := versionutil.ParseSemantic(p.Options.Version)
	if err != nil {
		return fmt.Errorf("failed to parse current version for the %s provider: %w", name, err)
	}

	releaseSeries := latestMetadata.GetReleaseSeriesForVersion(targetVersion)
	if releaseSeries == nil {
		return fmt.Errorf("invalid provider metadata: version %s for the provider %s does not match any release series", p.Options.Version, name)
	}

	if releaseSeries.Contract != "v1alpha4" && releaseSeries.Contract != "v1beta1" {
		return fmt.Errorf(capiVersionIncompatibilityMessage, clusterv1.GroupVersion.Version, releaseSeries.Contract, name)
	}

	p.Contract = releaseSeries.Contract

	return nil
}

// Fetch fetches the provider components from the repository and processes all yaml manifests.
func (p *PhaseReconciler[P, G]) Fetch(ctx context.Context, phase G) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching provider")

	// Fetch the provider components yaml file from the provided repository GitHub/GitLab/ConfigMap.
	componentsFile, err := p.Repo.GetFile(ctx, p.Options.Version, p.Repo.ComponentsPath())
	if err != nil {
		err = fmt.Errorf("failed to read %q from provider's repository %q: %w", p.Repo.ComponentsPath(), p.ProviderConfig.ManifestLabel(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Generate a set of new objects using the clusterctl library. NewComponents() will do the yaml processing,
	// like ensure all the provider components are in proper namespace, replace variables, etc. See the clusterctl
	// documentation for more details.
	p.Components, err = repository.NewComponents(repository.ComponentsInput{
		Provider:     p.ProviderConfig,
		ConfigClient: p.ConfigClient,
		Processor:    yamlprocessor.NewSimpleProcessor(),
		RawYaml:      componentsFile,
		Options:      p.Options,
	})
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// ProviderSpec provides fields for customizing the provider deployment options.
	// We can use clusterctl library to apply this customizations.
	if err := repository.AlterComponents(p.Components, customizeObjectsFn(phase.GetProvider())); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Apply patches to the provider components if specified.
	if err := repository.AlterComponents(p.Components, applyPatches(ctx, phase.GetProvider())); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	conditions.Set(phase.GetProvider(), conditions.TrueCondition(operatorv1.ProviderInstalledCondition))

	return reconcile.Result{}, nil
}

// Upgrade ensure all the clusterctl CRDs are available before installing the provider,
// and update existing components if required.
func (p *PhaseReconciler[P, G]) Upgrade(ctx context.Context, phase G) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Nothing to do if it's a fresh installation.
	if phase.GetProvider().GetStatus().InstalledVersion == nil {
		return reconcile.Result{}, nil
	}

	// Provider needs to be re-installed
	if *phase.GetProvider().GetStatus().InstalledVersion == phase.GetProvider().GetSpec().Version {
		return reconcile.Result{}, nil
	}

	log.Info("Version changes detected, updating existing components")

	if err := newClusterClient(p, phase).ProviderUpgrader().ApplyCustomPlan(ctx, cluster.UpgradeOptions{}, cluster.UpgradeItem{
		NextVersion: phase.GetProvider().GetSpec().Version,
		Provider:    setProviderVersion(phase.GetClusterctlProvider(), phase.GetProvider().GetSpec().Version),
	}); err != nil {
		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsUpgradeErrorReason, operatorv1.ProviderUpgradedCondition)
	}

	log.Info("Provider successfully upgraded")
	conditions.Set(phase.GetProvider(), conditions.TrueCondition(operatorv1.ProviderUpgradedCondition))

	return reconcile.Result{}, nil
}

// Install installs the provider components using clusterctl library.
func (p *PhaseReconciler[P, G]) Install(ctx context.Context, phase G) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Provider was upgraded, nothing to do
	if phase.GetProvider().GetStatus().InstalledVersion != nil && *phase.GetProvider().GetStatus().InstalledVersion != phase.GetProvider().GetSpec().Version {
		return reconcile.Result{}, nil
	}

	clusterClient := newClusterClient(p, phase)

	log.Info("Installing provider")

	if err := clusterClient.ProviderComponents().Create(ctx, p.Components.Objs()); err != nil {
		reason := "Install failed"
		if wait.Interrupted(err) {
			reason = "Timed out waiting for deployment to become ready"
		}

		return reconcile.Result{}, wrapPhaseError(err, reason, operatorv1.ProviderInstalledCondition)
	}

	log.Info("Provider successfully installed")
	conditions.Set(phase.GetProvider(), conditions.TrueCondition(operatorv1.ProviderInstalledCondition))

	return reconcile.Result{}, nil
}

func (p *PhaseReconciler[P, G]) ReportStatus(_ context.Context, phase G) (reconcile.Result, error) {
	status := phase.GetProvider().GetStatus()
	status.Contract = &p.Contract
	installedVersion := p.Components.Version()
	status.InstalledVersion = &installedVersion
	phase.GetProvider().SetStatus(status)

	return reconcile.Result{}, nil
}

func setProviderVersion(clusterctlProvider *clusterctlv1.Provider, version string) clusterctlv1.Provider {
	clusterctlProvider.Version = version

	return *clusterctlProvider
}

// Delete deletes the provider components using clusterctl library.
func (p *PhaseReconciler[P, G]) Delete(ctx context.Context, phase G) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting provider")

	clusterClient := newClusterClient(p, phase)

	err := clusterClient.ProviderComponents().Delete(ctx, cluster.DeleteOptions{
		Provider:         *phase.GetClusterctlProvider(),
		IncludeNamespace: false,
		IncludeCRDs:      false,
	})

	return reconcile.Result{}, wrapPhaseError(err, operatorv1.OldComponentsDeletionErrorReason, operatorv1.ProviderInstalledCondition)
}

func (p *PhaseReconciler[P, G]) repositoryProxy(ctx context.Context, provider configclient.Provider, configClient configclient.Client, options ...repository.Option) (repository.Client, error) {
	injectRepo := p.Repo

	if !provider.SameAs(p.ProviderConfig) {
		genericProvider, err := GetGenericProvider(ctx, p.ctrlClient, provider)
		if err != nil {
			return nil, wrapPhaseError(err, "unable to find generic provider for configclient "+string(provider.Type())+": "+provider.Name(), operatorv1.ProviderUpgradedCondition)
		}

		if exists, err := checkConfigMapExists(ctx, p.ctrlClient, *providerLabelSelector(genericProvider), genericProvider.GetNamespace()); err != nil {
			provider := client.ObjectKeyFromObject(genericProvider)
			return nil, wrapPhaseError(err, "failed to check the config map repository existence for provider "+provider.String(), operatorv1.ProviderUpgradedCondition)
		} else if !exists {
			provider := client.ObjectKeyFromObject(genericProvider)
			return nil, wrapPhaseError(fmt.Errorf("config map not found"), "config map repository required for validation does not exist yet for provider "+provider.String(), operatorv1.ProviderUpgradedCondition)
		}

		repo, err := configmapRepository(ctx, p.ctrlClient, providerLabelSelector(genericProvider), genericProvider.GetNamespace(), "")
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

	return proxy.RepositoryProxy{Client: cl, RepositoryComponents: p.Components}, nil
}

// GetGenericProvider returns the first of generic providers matching the type and the name from the configclient.Provider.
func GetGenericProvider(ctx context.Context, cl client.Client, provider configclient.Provider) (operatorv1.GenericProvider, error) {
	p, found := generic.ProviderReconcilers[provider.Type()]
	if !found {
		return nil, fmt.Errorf("provider %s type %s is not supported", provider.Name(), provider.Type())
	}

	list := p.GetProviderList()
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

// newClusterClient returns a clusterctl client for interacting with management cluster.
func newClusterClient[P generic.Provider, G generic.Group[P]](p *PhaseReconciler[P, G], phase G) cluster.Client {
	return cluster.New(cluster.Kubeconfig{}, p.ConfigClient, cluster.InjectProxy(&proxy.ControllerProxy{
		CtrlClient: proxy.ClientProxy{
			Client:        phase.GetClient(),
			ListProviders: listProviders,
		},
		CtrlConfig: phase.GetConfig(),
	}), cluster.InjectRepositoryFactory(p.repositoryProxy))
}

func listProviders(ctx context.Context, cl client.Client, list *clusterctlv1.ProviderList, opts ...client.ListOption) error {
	return kerrors.NewAggregate([]error{
		combineProviders[*operatorv1.CoreProvider](ctx, cl, list, opts...),
		combineProviders[*operatorv1.InfrastructureProvider](ctx, cl, list, opts...),
		combineProviders[*operatorv1.BootstrapProvider](ctx, cl, list, opts...),
		combineProviders[*operatorv1.ControlPlaneProvider](ctx, cl, list, opts...),
		combineProviders[*operatorv1.AddonProvider](ctx, cl, list, opts...),
		combineProviders[*operatorv1.IPAMProvider](ctx, cl, list, opts...),
	})
}

func combineProviders[P generic.Provider](ctx context.Context, cl client.Client, list *clusterctlv1.ProviderList, opts ...client.ListOption) error {
	reconciler := generic.GetBuilder(*new(P))
	if reconciler == nil {
		return fmt.Errorf("unable to find registered reconciler for type: %T", *new(P))
	}

	l := reconciler.GetProviderList()
	if err := cl.List(ctx, l, opts...); err != nil {
		return err
	}

	for _, p := range l.GetItems() {
		provider, ok := p.(P)
		if !ok {
			return fmt.Errorf("unexpected provider type: %T, expected %T", p, new(P))
		}

		list.Items = append(list.Items, *reconciler.ClusterctlProvider(provider))
	}

	return nil
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
