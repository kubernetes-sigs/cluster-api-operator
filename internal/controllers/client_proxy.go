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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// controllerProxy implements the Proxy interface from the clusterctl. It is used to
// interact with the management cluster.
type controllerProxy struct {
	ctrlClient client.Client
	ctrlConfig *rest.Config
}

var _ cluster.Proxy = &controllerProxy{}

func (k *controllerProxy) CurrentNamespace() (string, error)           { return "default", nil }
func (k *controllerProxy) ValidateKubernetesVersion() error            { return nil }
func (k *controllerProxy) GetConfig() (*rest.Config, error)            { return k.ctrlConfig, nil }
func (k *controllerProxy) NewClient() (client.Client, error)           { return k.ctrlClient, nil }
func (k *controllerProxy) GetContexts(prefix string) ([]string, error) { return nil, nil }
func (k *controllerProxy) CheckClusterAvailable() error                { return nil }

// GetResourceNames returns the list of resource names which begin with prefix.
func (k *controllerProxy) GetResourceNames(groupVersion, kind string, options []client.ListOption, prefix string) ([]string, error) {
	objList, err := listObjByGVK(k.ctrlClient, groupVersion, kind, options)
	if err != nil {
		return nil, err
	}

	var comps []string

	for _, item := range objList.Items {
		name := item.GetName()

		if strings.HasPrefix(name, prefix) {
			comps = append(comps, name)
		}
	}

	return comps, nil
}

// ListResources lists namespaced and cluster-wide resources for a component matching the labels.
func (k *controllerProxy) ListResources(labels map[string]string, namespaces ...string) ([]unstructured.Unstructured, error) {
	resourceList := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Kind: "Secret", Namespaced: true},
				{Kind: "ConfigMap", Namespaced: true},
				{Kind: "Service", Namespaced: true},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Kind: "DaemonSet", Namespaced: true},
				{Kind: "Deployment", Namespaced: true},
			},
		},
		{
			GroupVersion: "admissionregistration.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Kind: "ValidatingWebhookConfiguration", Namespaced: true},
				{Kind: "MutatingWebhookConfiguration", Namespaced: true},
			},
		},
	}

	var ret []unstructured.Unstructured

	for _, resourceGroup := range resourceList {
		for _, resourceKind := range resourceGroup.APIResources {
			if resourceKind.Namespaced {
				for _, namespace := range namespaces {
					objList, err := listObjByGVK(k.ctrlClient, resourceGroup.GroupVersion, resourceKind.Kind, []client.ListOption{client.MatchingLabels(labels), client.InNamespace(namespace)})
					if err != nil {
						return nil, err
					}

					klog.V(3).InfoS("listed", "kind", resourceKind.Kind, "count", len(objList.Items))

					ret = append(ret, objList.Items...)
				}
			} else {
				objList, err := listObjByGVK(k.ctrlClient, resourceGroup.GroupVersion, resourceKind.Kind, []client.ListOption{client.MatchingLabels(labels)})
				if err != nil {
					return nil, err
				}
				klog.V(3).InfoS("listed", "kind", resourceKind.Kind, "count", len(objList.Items))
				ret = append(ret, objList.Items...)
			}
		}
	}

	return ret, nil
}

func listObjByGVK(c client.Client, groupVersion, kind string, options []client.ListOption) (*unstructured.UnstructuredList, error) {
	ctx := context.TODO()
	objList := new(unstructured.UnstructuredList)
	objList.SetAPIVersion(groupVersion)
	objList.SetKind(kind)

	if err := c.List(ctx, objList, options...); err != nil {
		if !errors.Is(err, &meta.NoKindMatchError{}) {
			return nil, fmt.Errorf("failed to list objects for the %q GroupVersionKind: %w", objList.GroupVersionKind(), err)
		}
	}

	return objList, nil
}
