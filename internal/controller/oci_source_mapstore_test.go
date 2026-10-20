/*
Copyright 2025 The Kubernetes Authors.

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
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

func testProvider(name, version string, manifestName *operatorv1.OCIManifestName) *operatorv1.InfrastructureProvider {
	return &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: version,
				FetchConfig: &operatorv1.FetchConfiguration{
					OCIConfiguration: operatorv1.OCIConfiguration{
						OCI:             "registry.example.com/provider:v0.1.0",
						OCIManifestName: manifestName,
					},
				},
			},
		},
	}
}

func Test_mapStore_defaultFileNames(t *testing.T) {
	g := NewWithT(t)

	p := testProvider("openstack", "v0.15.0", nil)
	store := NewMapStore(p)

	store.data["infrastructure-openstack-v0.15.0-metadata.yaml"] = []byte("metadata")
	store.data["infrastructure-openstack-v0.15.0-components.yaml"] = []byte("components")

	metadata, err := store.GetMetadata(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metadata).To(Equal([]byte("metadata")))

	components, err := store.GetComponents(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(components).To(Equal([]byte("components")))
}

func Test_mapStore_defaultFileFallbacks(t *testing.T) {
	g := NewWithT(t)

	p := testProvider("custom-name", "v0.15.0", nil)
	store := NewMapStore(p)

	// Only generic files are provided, default lookup order applies.
	store.data["metadata.yaml"] = []byte("metadata")
	store.data["infrastructure-components.yaml"] = []byte("components")

	metadata, err := store.GetMetadata(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metadata).To(Equal([]byte("metadata")))

	components, err := store.GetComponents(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(components).To(Equal([]byte("components")))
}

func Test_mapStore_missingFiles(t *testing.T) {
	g := NewWithT(t)

	p := testProvider("custom-name", "v0.15.0", nil)
	store := NewMapStore(p)

	_, err := store.GetMetadata(p)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("infrastructure-custom-name-v0.15.0-metadata.yaml"))

	_, err = store.GetComponents(p)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("infrastructure-custom-name-v0.15.0-components.yaml"))
}

func Test_mapStore_OCIManifestNameOverride(t *testing.T) {
	g := NewWithT(t)

	p := testProvider("supplemental-openstack", "v0.15.0", &operatorv1.OCIManifestName{
		Metadata:   "infrastructure-openstack-v0.15.0-metadata.yaml",
		Components: "infrastructure-openstack-v0.15.0-components.yaml",
	})

	store := NewMapStore(p)

	// Override file names must be registered for the OCI copy to store them.
	g.Expect(store.data).To(HaveKey("infrastructure-openstack-v0.15.0-metadata.yaml"))
	g.Expect(store.data).To(HaveKey("infrastructure-openstack-v0.15.0-components.yaml"))

	store.data["infrastructure-openstack-v0.15.0-metadata.yaml"] = []byte("metadata")
	store.data["infrastructure-openstack-v0.15.0-components.yaml"] = []byte("components")

	// Override file names take precedence over the default derived names.
	metadata, err := store.GetMetadata(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metadata).To(Equal([]byte("metadata")))

	components, err := store.GetComponents(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(components).To(Equal([]byte("components")))
}

func Test_mapStore_OCIManifestNamePartialOverride(t *testing.T) {
	g := NewWithT(t)

	// Only the metadata name is overridden, components keep the default lookup.
	p := testProvider("supplemental-openstack", "v0.15.0", &operatorv1.OCIManifestName{
		Metadata: "infrastructure-openstack-v0.15.0-metadata.yaml",
	})

	store := NewMapStore(p)
	store.data["infrastructure-openstack-v0.15.0-metadata.yaml"] = []byte("metadata")
	store.data["infrastructure-supplemental-openstack-v0.15.0-components.yaml"] = []byte("components")

	metadata, err := store.GetMetadata(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(metadata).To(Equal([]byte("metadata")))

	components, err := store.GetComponents(p)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(components).To(Equal([]byte("components")))
}

func Test_mapStore_OCIManifestNameMissingOverrideFile(t *testing.T) {
	g := NewWithT(t)

	p := testProvider("supplemental-openstack", "v0.15.0", &operatorv1.OCIManifestName{
		Metadata:   "infrastructure-openstack-v0.15.0-metadata.yaml",
		Components: "infrastructure-openstack-v0.15.0-components.yaml",
	})

	store := NewMapStore(p)

	// Default names must not be used when an explicit override misses the file.
	_, err := store.GetMetadata(p)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("collected artifact needs to provide metadata as infrastructure-openstack-v0.15.0-metadata.yaml file"))

	_, err = store.GetComponents(p)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("collected artifact needs to provide components as infrastructure-openstack-v0.15.0-components.yaml file"))
}
