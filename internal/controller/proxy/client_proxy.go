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

package proxy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// kerrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	// "sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClientProxy implements the Proxy interface from the clusterctl. It is used to
// interact with the management cluster.
type ClientProxy struct {
	client.Client
	ListProviders func(context.Context, client.Client, *clusterctlv1.ProviderList, ...client.ListOption) error
}

func (c ClientProxy) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	switch l := list.(type) {
	case *clusterctlv1.ProviderList:
		return c.ListProviders(ctx, c.Client, l, opts...)
	default:
		return c.Client.List(ctx, l, opts...)
	}
}

func (c ClientProxy) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	switch o := obj.(type) {
	case *clusterctlv1.Provider:
		return nil
	default:
		return c.Client.Get(ctx, key, o, opts...)
	}
}

func (c ClientProxy) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	switch o := obj.(type) {
	case *clusterctlv1.Provider:
		return nil
	default:
		return c.Client.Patch(ctx, o, patch, opts...)
	}
}

// ControllerProxy implements the Proxy interface from the clusterctl. It is used to
// interact with the management cluster.
type ControllerProxy struct {
	CtrlClient ClientProxy
	CtrlConfig *rest.Config
}

var _ cluster.Proxy = &ControllerProxy{}

func (k *ControllerProxy) CurrentNamespace() (string, error)           { return "default", nil }
func (k *ControllerProxy) ValidateKubernetesVersion() error            { return nil }
func (k *ControllerProxy) GetConfig() (*rest.Config, error)            { return k.CtrlConfig, nil }
func (k *ControllerProxy) NewClient() (client.Client, error)           { return k.CtrlClient, nil }
func (k *ControllerProxy) GetContexts(prefix string) ([]string, error) { return nil, nil }
func (k *ControllerProxy) CheckClusterAvailable() error                { return nil }

// GetResourceNames returns the list of resource names which begin with prefix.
func (k *ControllerProxy) GetResourceNames(ctx context.Context, groupVersion, kind string, options []client.ListOption, prefix string) ([]string, error) {
	objList, err := listObjByGVK(ctx, k.CtrlClient, groupVersion, kind, options)
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
func (k *ControllerProxy) ListResources(ctx context.Context, labels map[string]string, namespaces ...string) ([]unstructured.Unstructured, error) {
	resourceList := []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Kind: "Secret", Namespaced: true},
				{Kind: "ConfigMap", Namespaced: true},
				{Kind: "Service", Namespaced: true},
				{Kind: "ServiceAccount", Namespaced: true},
				{Kind: "Namespace"},
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
				{Kind: "ValidatingWebhookConfiguration"},
				{Kind: "MutatingWebhookConfiguration"},
			},
		},
		{
			GroupVersion: "apiextensions.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Kind: "CustomResourceDefinition"},
			},
		},
		{
			GroupVersion: "rbac.authorization.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Kind: "Role", Namespaced: true},
				{Kind: "RoleBinding", Namespaced: true},
				{Kind: "ClusterRole"},
				{Kind: "ClusterRoleBinding"},
			},
		},
	}

	var ret []unstructured.Unstructured

	for _, resourceGroup := range resourceList {
		for _, resourceKind := range resourceGroup.APIResources {
			if resourceKind.Namespaced {
				for _, namespace := range namespaces {
					objList, err := listObjByGVK(ctx, k.CtrlClient, resourceGroup.GroupVersion, resourceKind.Kind, []client.ListOption{client.MatchingLabels(labels), client.InNamespace(namespace)})
					if err != nil {
						return nil, err
					}

					klog.V(3).InfoS("listed", "kind", resourceKind.Kind, "count", len(objList.Items))

					ret = append(ret, objList.Items...)
				}
			} else {
				objList, err := listObjByGVK(ctx, k.CtrlClient, resourceGroup.GroupVersion, resourceKind.Kind, []client.ListOption{client.MatchingLabels(labels)})
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

func listObjByGVK(ctx context.Context, c client.Client, groupVersion, kind string, options []client.ListOption) (*unstructured.UnstructuredList, error) {
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

type RepositoryProxy struct {
	repository.Client

	RepositoryComponents repository.Components
}

type RepositoryClient struct {
	repository.Components
}

func (r RepositoryClient) Raw(ctx context.Context, options repository.ComponentsOptions) ([]byte, error) {
	return nil, nil
}

func (r RepositoryClient) Get(ctx context.Context, options repository.ComponentsOptions) (repository.Components, error) {
	return r.Components, nil
}

func (r RepositoryProxy) Components() repository.ComponentsClient {
	return RepositoryClient{r.RepositoryComponents}
}
