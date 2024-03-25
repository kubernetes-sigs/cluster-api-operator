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
	"context"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
)

func TestManifestsDownloader(t *testing.T) {
	g := NewWithT(t)

	ctx := context.Background()

	fakeclient := fake.NewClientBuilder().WithObjects().Build()

	namespace := "test-namespace"

	core := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-api",
			Namespace: namespace,
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v1.4.3",
			},
		},
	}

	p := &PhaseReconciler[*operatorv1.CoreProvider, Phase[*operatorv1.CoreProvider]]{
		ctrlClient: fakeclient,
	}

	_, err := p.InitializePhaseReconciler(ctx, Phase[*operatorv1.CoreProvider]{
		Provider:     core,
		ProviderType: clusterctlv1.CoreProviderType,
		ProviderList: &operatorv1.CoreProviderList{},
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = p.DownloadManifests(ctx, Phase[*operatorv1.CoreProvider]{
		Client:       fakeclient,
		Provider:     core,
		ProviderType: clusterctlv1.CoreProviderType,
		ProviderList: &operatorv1.CoreProviderList{},
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Ensure that config map was created
	labelSelector := metav1.LabelSelector{
		MatchLabels: providerLabels(core),
	}

	exists, err := checkConfigMapExists(ctx, fakeclient, labelSelector, core.GetNamespace())
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(exists).To(BeTrue())
}
