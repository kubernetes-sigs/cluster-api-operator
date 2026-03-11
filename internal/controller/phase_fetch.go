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
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apijson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/yamlprocessor"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Fetch fetches the provider components from the repository and processes all yaml manifests.
func (p *PhaseReconciler) Fetch(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching provider")

	// Fetch the provider components yaml file from the provided repository GitHub/GitLab/ConfigMap.
	componentsFile, err := p.repo.GetFile(ctx, p.options.Version, p.repo.ComponentsPath())
	if err != nil {
		err = fmt.Errorf("failed to read %q from provider's repository %q: %w", p.repo.ComponentsPath(), p.providerConfig.ManifestLabel(), err)

		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Check if components exceed the resource size.
	p.needsCompression = needToCompress(componentsFile)

	log.V(2).Info("Fetched components file", "size", len(componentsFile), "needsCompression", p.needsCompression)

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
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsFetchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// ProviderSpec provides fields for customizing the provider deployment options.
	// We can use clusterctl library to apply this customizations.
	if err := repository.AlterComponents(p.components, customizeObjectsFn(p.provider)); err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Apply patches to the provider components if specified.
	if err := repository.AlterComponents(p.components, applyPatches(ctx, p.provider)); err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsPatchErrorReason, operatorv1.ProviderInstalledCondition)
	}

	// Apply image overrides to the provider manifests.
	if err := repository.AlterComponents(p.components, imageOverrides(p.components.ManifestLabel(), p.overridesClient)); err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsImageOverrideErrorReason, operatorv1.ProviderInstalledCondition)
	}

	for _, fn := range p.customAlterComponentsFuncs {
		if err := repository.AlterComponents(p.components, fn); err != nil {
			return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
		}
	}

	return &Result{}, nil
}

// Store stores the provider components in the cache.
func (p *PhaseReconciler) Store(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Storing provider in cache")

	kinds, _, err := scheme.Scheme.ObjectKinds(&corev1.Secret{})
	if err != nil || len(kinds) == 0 {
		err = fmt.Errorf("cannot fetch kind of the Secret resource: %w", err)

		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds[0].Kind,
			APIVersion: kinds[0].GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ProviderCacheName(p.provider),
			Namespace:   p.provider.GetNamespace(),
			Annotations: map[string]string{},
		},
		StringData: map[string]string{},
		Data:       map[string][]byte{},
	}

	gvk := p.provider.GetObjectKind().GroupVersionKind()

	secret.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       p.provider.GetName(),
			UID:        p.provider.GetUID(),
		},
	})

	if p.needsCompression {
		secret.Annotations[operatorv1.CompressedAnnotation] = "true"
	}

	manifests, err := apijson.Marshal(addNamespaceIfMissing(p.components.Objs(), p.provider.GetNamespace()))
	if err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
	}

	if p.needsCompression {
		var buf bytes.Buffer
		if err := compressData(&buf, manifests); err != nil {
			return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
		}

		secret.Data["cache"] = buf.Bytes()
	} else {
		secret.StringData["cache"] = string(manifests)
	}

	if err := p.ctrlClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(cacheOwner)); err != nil {
		return &Result{}, wrapPhaseError(err, operatorv1.ComponentsCustomizationErrorReason, operatorv1.ProviderInstalledCondition)
	}

	return &Result{}, nil
}

// addNamespaceIfMissing adda a Namespace object if missing (this ensure the targetNamespace will be created).
func addNamespaceIfMissing(objs []unstructured.Unstructured, targetNamespace string) []unstructured.Unstructured {
	namespaceObjectFound := false

	for _, o := range objs {
		// if the object has Kind Namespace, fix the namespace name
		if o.GetKind() == namespaceKind {
			namespaceObjectFound = true
		}
	}

	// if there isn't an object with Kind Namespace, add it
	if !namespaceObjectFound {
		objs = append(objs, unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       namespaceKind,
				"metadata": map[string]interface{}{
					"name": targetNamespace,
				},
			},
		})
	}

	return objs
}

func (p *PhaseReconciler) ReportStatus(ctx context.Context) (*Result, error) {
	log := ctrl.LoggerFrom(ctx)

	status := p.provider.GetStatus()
	status.Contract = &p.contract
	installedVersion := p.components.Version()
	status.InstalledVersion = &installedVersion
	p.provider.SetStatus(status)

	log.V(2).Info("Reported provider status", "contract", p.contract, "installedVersion", installedVersion)

	return &Result{}, nil
}
