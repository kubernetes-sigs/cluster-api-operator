/*
Copyright 2024 The Kubernetes Authors.

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

package providers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/ptr"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/cluster-api-operator/util"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

type IPAMProviderReconciler struct {
	generic.ProviderReconciler[*operatorv1.IPAMProvider]
}

func NewIPAMProviderReconciler(conn generic.Connector) generic.ProviderReconciler[*operatorv1.IPAMProvider] {
	return &IPAMProviderReconciler{
		ProviderReconciler: NewCommonProviderReconciler[*operatorv1.IPAMProvider](conn),
	}
}

// ClusterctlProviderType returns ProviderType for the underlying clusterctl provider
func (r *IPAMProviderReconciler) ClusterctlProviderType() clusterctlv1.ProviderType {
	return clusterctlv1.IPAMProviderType
}

// ClusterctlProvider returns Provider stucture of the underlying clusterctl provider
func (r *IPAMProviderReconciler) ClusterctlProvider(provider *operatorv1.IPAMProvider) *clusterctlv1.Provider {
	clusterctlProvider := &clusterctlv1.Provider{ObjectMeta: metav1.ObjectMeta{
		Name:      "ipam-" + provider.GetName(),
		Namespace: provider.GetNamespace(),
	},
		Type:         string(r.ClusterctlProviderType()),
		ProviderName: provider.GetName(),
		Version:      *util.Or(provider.GetStatus().InstalledVersion, ptr.To("")),
	}

	return clusterctlProvider
}

// ProviderList returns empty typed list for provider
func (r *IPAMProviderReconciler) GetProviderList() generic.ProviderList {
	return &operatorv1.IPAMProviderList{}
}
