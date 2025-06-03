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

package controller

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"oras.land/oras-go/v2/registry/remote/auth"

	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
)

const (
	configMapSourceLabel      = "provider.cluster.x-k8s.io/source"
	configMapSourceAnnotation = "provider.cluster.x-k8s.io/source"
	operatorManagedLabel      = "managed-by.operator.cluster.x-k8s.io"

	maxConfigMapSize = 1 * 1024 * 1024
	ociSource        = "oci"
)

// DownloadManifests downloads CAPI manifests from a url.
func (p *PhaseReconciler) DownloadManifests(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Return immediately if a custom config map is used instead of a url.
	if p.provider.GetSpec().FetchConfig != nil && p.provider.GetSpec().FetchConfig.Selector != nil {
		log.V(5).Info("Custom config map is used, skip downloading provider manifests")

		return &Result{}, nil
	}

	// Check if manifests are already downloaded and stored in a configmap
	labelSelector := metav1.LabelSelector{
		MatchLabels: p.prepareConfigMapLabels(),
	}

	exists, err := p.checkConfigMapExists(ctx, labelSelector, p.provider.GetNamespace())
	if err != nil {
		return &Result{}, wrapPhaseError(err, "failed to check that config map with manifests exists", operatorv1.ProviderInstalledCondition)
	}

	if exists {
		log.V(5).Info("Config map with downloaded manifests already exists, skip downloading provider manifests")

		return &Result{}, nil
	}

	log.Info("Downloading provider manifests")

	if p.providerConfig.URL() != fakeURL {
		p.repo, err = util.RepositoryFactory(ctx, p.providerConfig, p.configClient.Variables())
		if err != nil {
			err = fmt.Errorf("failed to create repo from provider url for provider %q: %w", p.provider.GetName(), err)

			return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
		}
	}

	spec := p.provider.GetSpec()

	if spec.Version == "" && p.repo != nil {
		// User didn't set the version, try to get repository default.
		spec.Version = p.repo.DefaultVersion()

		// Add version to the provider spec.
		p.provider.SetSpec(spec)
	}

	var configMap *corev1.ConfigMap

	// Fetch the provider metadata and components yaml files from the provided repository GitHub/GitLab or OCI source
	if p.provider.GetSpec().FetchConfig != nil && p.provider.GetSpec().FetchConfig.OCI != "" {
		configMap, err = OCIConfigMap(ctx, p.provider, OCIAuthentication(p.configClient.Variables()))
		if err != nil {
			return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
		}
	} else {
		configMap, err = RepositoryConfigMap(ctx, p.provider, p.repo)
		if err != nil {
			err = fmt.Errorf("failed to create config map for provider %q: %w", p.provider.GetName(), err)

			return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
		}
	}

	if err := p.ctrlClient.Create(ctx, configMap); client.IgnoreAlreadyExists(err) != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	return &Result{}, nil
}

// checkConfigMapExists checks if a config map exists in Kubernetes with the given LabelSelector.
func (p *PhaseReconciler) checkConfigMapExists(ctx context.Context, labelSelector metav1.LabelSelector, namespace string) (bool, error) {
	labelSet := labels.Set(labelSelector.MatchLabels)
	listOpts := []client.ListOption{
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(labelSet)},
		client.InNamespace(namespace),
	}

	var configMapList corev1.ConfigMapList

	if err := p.ctrlClient.List(ctx, &configMapList, listOpts...); err != nil {
		return false, fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	if len(configMapList.Items) > 1 {
		return false, fmt.Errorf("more than one config maps were found for given selector: %v", labelSelector.String())
	}

	return len(configMapList.Items) == 1, nil
}

// prepareConfigMapLabels returns labels that identify a config map with downloaded manifests.
func (p *PhaseReconciler) prepareConfigMapLabels() map[string]string {
	return ProviderLabels(p.provider)
}

// TemplateManifestsConfigMap prepares a config map with downloaded manifests.
func TemplateManifestsConfigMap(provider operatorv1.GenericProvider, labels map[string]string, metadata, components []byte, compress bool) (*corev1.ConfigMap, error) {
	configMapName := fmt.Sprintf("%s-%s-%s", provider.GetType(), provider.GetName(), provider.GetSpec().Version)

	kinds, _, err := clientgoscheme.Scheme.ObjectKinds(&corev1.ConfigMap{})
	if err != nil || len(kinds) == 0 {
		return nil, fmt.Errorf("cannot fetch kind of the ConfigMap resource: %w", err)
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds[0].Kind,
			APIVersion: kinds[0].GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: provider.GetNamespace(),
			Labels:    labels,
		},
		Data: map[string]string{
			operatorv1.MetadataConfigMapKey: string(metadata),
		},
	}

	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.OCI != "" {
		configMap.ObjectMeta.Annotations = map[string]string{
			configMapSourceAnnotation: provider.GetSpec().FetchConfig.OCI,
		}
	}

	// Components manifests data can exceed the configmap size limit. In this case we have to compress it.
	if !compress {
		configMap.Data[operatorv1.ComponentsConfigMapKey] = string(components)
	} else {
		var componentsBuf bytes.Buffer
		zw := gzip.NewWriter(&componentsBuf)

		_, err := zw.Write(components)
		if err != nil {
			return nil, fmt.Errorf("cannot compress data for provider %s/%s: %w", provider.GetNamespace(), provider.GetName(), err)
		}

		if err := zw.Close(); err != nil {
			return nil, err
		}

		configMap.BinaryData = map[string][]byte{
			operatorv1.ComponentsConfigMapKey: componentsBuf.Bytes(),
		}

		// Setting the annotation to mark these manifests as compressed.
		configMap.SetAnnotations(map[string]string{operatorv1.CompressedAnnotation: "true"})
	}

	gvk := provider.GetObjectKind().GroupVersionKind()

	configMap.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       provider.GetName(),
			UID:        provider.GetUID(),
		},
	})

	return configMap, nil
}

// OCIConfigMap templates config from the OCI source.
func OCIConfigMap(ctx context.Context, provider operatorv1.GenericProvider, auth *auth.Credential) (*corev1.ConfigMap, error) {
	store, err := FetchOCI(ctx, provider, auth)
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

	configMap, err := TemplateManifestsConfigMap(provider, ProviderLabels(provider), metadata, components, needToCompress(metadata, components))
	if err != nil {
		err = fmt.Errorf("failed to create config map for provider %q: %w", provider.GetName(), err)

		return nil, err
	}

	if provider.GetUID() == "" {
		// Unset owner references due to lack of existing provider owner object
		configMap.OwnerReferences = nil
	}

	return configMap, nil
}

// RepositoryConfigMap templates ConfigMap resource from the provider repository.
func RepositoryConfigMap(ctx context.Context, provider operatorv1.GenericProvider, repo repository.Repository) (*corev1.ConfigMap, error) {
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

	configMap, err := TemplateManifestsConfigMap(provider, ProviderLabels(provider), metadata, components, needToCompress(metadata, components))
	if err != nil {
		err = fmt.Errorf("failed to create config map for provider %q: %w", provider.GetName(), err)

		return nil, err
	}

	if provider.GetUID() == "" {
		// Unset owner references due to lack of existing provider owner object
		configMap.OwnerReferences = nil
	}

	return configMap, nil
}

func providerLabelSelector(provider operatorv1.GenericProvider) *metav1.LabelSelector {
	// Replace label selector if user wants to use custom config map
	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.Selector != nil {
		return provider.GetSpec().FetchConfig.Selector
	}

	return &metav1.LabelSelector{
		MatchLabels: ProviderLabels(provider),
	}
}

// ProviderLabels returns default set of labels that identify a config map with downloaded manifests.
func ProviderLabels(provider operatorv1.GenericProvider) map[string]string {
	labels := map[string]string{
		operatorv1.ConfigMapVersionLabelName: provider.GetSpec().Version,
		operatorv1.ConfigMapTypeLabel:        provider.GetType(),
		operatorv1.ConfigMapNameLabel:        provider.GetName(),
		operatorManagedLabel:                 "true",
	}

	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.OCI != "" {
		labels[configMapSourceLabel] = ociSource
	}

	return labels
}

// needToCompress checks whether the input data exceeds the maximum configmap
// size limit and returns whether it should be compressed.
func needToCompress(bs ...[]byte) bool {
	totalBytes := 0

	for _, b := range bs {
		totalBytes += len(b)
	}

	return totalBytes > maxConfigMapSize
}
