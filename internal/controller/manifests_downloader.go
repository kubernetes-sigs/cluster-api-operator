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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/util"
)

const (
	configMapVersionLabel = "provider.cluster.x-k8s.io/version"
	configMapTypeLabel    = "provider.cluster.x-k8s.io/type"
	configMapNameLabel    = "provider.cluster.x-k8s.io/name"
	operatorManagedLabel  = "managed-by.operator.cluster.x-k8s.io"

	compressedAnnotation = "provider.cluster.x-k8s.io/compressed"

	metadataConfigMapKey            = "metadata"
	componentsConfigMapKey          = "components"
	additionalManifestsConfigMapKey = "manifests"

	maxConfigMapSize = 1 * 1024 * 1024
)

// downloadManifests downloads CAPI manifests from a url.
func (p *phaseReconciler) downloadManifests(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Return immediately if a custom config map is used instead of a url.
	if p.provider.GetSpec().FetchConfig != nil && p.provider.GetSpec().FetchConfig.Selector != nil {
		log.V(5).Info("Custom config map is used, skip downloading provider manifests")

		return reconcile.Result{}, nil
	}

	// Check if manifests are already downloaded and stored in a configmap
	labelSelector := metav1.LabelSelector{
		MatchLabels: p.prepareConfigMapLabels(),
	}

	exists, err := p.checkConfigMapExists(ctx, labelSelector, p.provider.GetNamespace())
	if err != nil {
		return reconcile.Result{}, wrapPhaseError(err, "failed to check that config map with manifests exists", operatorv1.ProviderInstalledCondition)
	}

	if exists {
		log.V(5).Info("Config map with downloaded manifests already exists, skip downloading provider manifests")

		return reconcile.Result{}, nil
	}

	log.Info("Downloading provider manifests")

	repo, err := util.RepositoryFactory(ctx, p.providerConfig, p.configClient.Variables())
	if err != nil {
		err = fmt.Errorf("failed to create repo from provider url for provider %q: %w", p.provider.GetName(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	spec := p.provider.GetSpec()

	if spec.Version == "" {
		// User didn't set the version, try to get repository default.
		spec.Version = repo.DefaultVersion()

		// Add version to the provider spec.
		p.provider.SetSpec(spec)
	}

	// Fetch the provider metadata and components yaml files from the provided repository GitHub/GitLab.
	metadataFile, err := repo.GetFile(ctx, spec.Version, metadataFile)
	if err != nil {
		err = fmt.Errorf("failed to read %q from the repository for provider %q: %w", metadataFile, p.provider.GetName(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	componentsFile, err := repo.GetFile(ctx, spec.Version, repo.ComponentsPath())
	if err != nil {
		err = fmt.Errorf("failed to read %q from the repository for provider %q: %w", componentsFile, p.provider.GetName(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	withCompression := needToCompress(metadataFile, componentsFile)

	if err := p.createManifestsConfigMap(ctx, metadataFile, componentsFile, withCompression); err != nil {
		err = fmt.Errorf("failed to create config map for provider %q: %w", p.provider.GetName(), err)

		return reconcile.Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	return reconcile.Result{}, nil
}

// checkConfigMapExists checks if a config map exists in Kubernetes with the given LabelSelector.
func (p *phaseReconciler) checkConfigMapExists(ctx context.Context, labelSelector metav1.LabelSelector, namespace string) (bool, error) {
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
func (p *phaseReconciler) prepareConfigMapLabels() map[string]string {
	return providerLabels(p.provider)
}

// createManifestsConfigMap creates a config map with downloaded manifests.
func (p *phaseReconciler) createManifestsConfigMap(ctx context.Context, metadata, components []byte, compress bool) error {
	configMapName := fmt.Sprintf("%s-%s-%s", p.provider.GetType(), p.provider.GetName(), p.provider.GetSpec().Version)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: p.provider.GetNamespace(),
			Labels:    p.prepareConfigMapLabels(),
		},
		Data: map[string]string{
			metadataConfigMapKey: string(metadata),
		},
	}

	// Components manifests data can exceed the configmap size limit. In this case we have to compress it.
	if !compress {
		configMap.Data[componentsConfigMapKey] = string(components)
	} else {
		var componentsBuf bytes.Buffer
		zw := gzip.NewWriter(&componentsBuf)

		_, err := zw.Write(components)
		if err != nil {
			return fmt.Errorf("cannot compress data for provider %s/%s: %w", p.provider.GetNamespace(), p.provider.GetName(), err)
		}

		if err := zw.Close(); err != nil {
			return err
		}

		configMap.BinaryData = map[string][]byte{
			componentsConfigMapKey: componentsBuf.Bytes(),
		}

		// Setting the annotation to mark these manifests as compressed.
		configMap.SetAnnotations(map[string]string{compressedAnnotation: "true"})
	}

	gvk := p.provider.GetObjectKind().GroupVersionKind()

	configMap.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       p.provider.GetName(),
			UID:        p.provider.GetUID(),
		},
	})

	if err := p.ctrlClient.Create(ctx, configMap); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func providerLabelSelector(provider operatorv1.GenericProvider) *metav1.LabelSelector {
	// Replace label selector if user wants to use custom config map
	if provider.GetSpec().FetchConfig != nil && provider.GetSpec().FetchConfig.Selector != nil {
		return provider.GetSpec().FetchConfig.Selector
	}

	return &metav1.LabelSelector{
		MatchLabels: providerLabels(provider),
	}
}

// prepareConfigMapLabels returns default set of labels that identify a config map with downloaded manifests.
func providerLabels(provider operatorv1.GenericProvider) map[string]string {
	return map[string]string{
		configMapVersionLabel: provider.GetSpec().Version,
		configMapTypeLabel:    provider.GetType(),
		configMapNameLabel:    provider.GetName(),
		operatorManagedLabel:  "true",
	}
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
