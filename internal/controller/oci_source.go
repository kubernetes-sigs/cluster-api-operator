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
	"context"
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ociUsernameKey     = "OCI_USERNAME"
	ociPasswordKey     = "OCI_PASSWORD"
	ociAccessTokenKey  = "OCI_ACCESS_TOKEN"
	ociRefreshTokenKey = "OCI_REFRESH_TOKEN" // #nosec G101

	metadataFile     = "metadata.yaml"
	fullMetadataFile = "%s-%s-%s-metadata.yaml"

	componentsFile      = "components.yaml"
	typedComponentsFile = "%s-components.yaml"
	fullComponentsFile  = "%s-%s-%s-components.yaml"
)

// mapStore is a pre-initialized map with expected file names to copy from OCI artifact.
type mapStore map[string][]byte

// NewMapStore initializes mapStore for the provider resource.
func NewMapStore(p operatorv1.GenericProvider) mapStore {
	return mapStore{
		metadataFile:   nil,
		componentsFile: nil,
		fmt.Sprintf(typedComponentsFile, p.GetType()):                                  nil,
		fmt.Sprintf(fullMetadataFile, p.GetType(), p.GetName(), p.GetSpec().Version):   nil,
		fmt.Sprintf(fullComponentsFile, p.GetType(), p.GetName(), p.GetSpec().Version): nil,
	}
}

// GetMetadata returns metadata file for the provider.
func (m mapStore) GetMetadata(p operatorv1.GenericProvider) ([]byte, error) {
	fullMetadataKey := fmt.Sprintf(fullMetadataFile, p.GetType(), p.GetName(), p.GetSpec().Version)

	data := m[fullMetadataKey]
	if len(data) != 0 {
		return data, nil
	}

	data = m[metadataFile]
	if len(data) != 0 {
		return data, nil
	}

	return nil, fmt.Errorf("collected artifact needs to provide metadata as %s or %s file", fullMetadataKey, metadataFile)
}

// GetComponents returns componenents file for the provider.
func (m mapStore) GetComponents(p operatorv1.GenericProvider) ([]byte, error) {
	fullComponentsKey := fmt.Sprintf(fullComponentsFile, p.GetType(), p.GetName(), p.GetSpec().Version)

	data := m[fullComponentsKey]
	if len(data) != 0 {
		return data, nil
	}

	typedComponentsKey := fmt.Sprintf(typedComponentsFile, p.GetType())

	data = m[typedComponentsKey]
	if len(data) != 0 {
		return data, nil
	}

	data = m[componentsFile]
	if len(data) != 0 {
		return data, nil
	}

	return nil, fmt.Errorf("collected artifact needs to provide components as %s or %s or %s file", fullComponentsKey, typedComponentsKey, componentsFile)
}

// selector is a PreCopy implementation for the oras.Target which fetches only expected files.
func (m mapStore) selector(_ context.Context, desc ocispec.Descriptor) error {
	file := desc.Annotations[ocispec.AnnotationTitle]
	if _, expected := m[file]; expected {
		return nil
	}

	return oras.SkipNode
}

// Exists implements oras.Target.
func (m mapStore) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return false, nil
}

// Fetch implements oras.Target.
func (m mapStore) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, nil
}

// Push implements oras.Target.
func (m mapStore) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) (err error) {
	// Verify we only store expected artifact names
	file := expected.Annotations[ocispec.AnnotationTitle]
	if _, expected := m[file]; expected {
		m[file], err = io.ReadAll(content)
	}

	return err
}

// Resolve implements oras.Target.
func (m mapStore) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	return ocispec.Descriptor{}, nil
}

// Tag implements oras.Target.
func (m mapStore) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	return nil
}

var _ oras.Target = &mapStore{}

// CopyOCIStore collects artifacts from the provider OCI url and creates a map of file contents.
func CopyOCIStore(ctx context.Context, url string, version string, store *mapStore, credential *auth.Credential) error {
	log := log.FromContext(ctx)

	if parts := strings.SplitN(url, ":", 2); len(parts) == 2 {
		url = parts[0]
		version = parts[1]
	}

	repo, err := remote.NewRepository(url)
	if err != nil {
		log.Error(err, "Invalid registry URL specified")

		return err
	}

	if credential != nil {
		repo.Client = &auth.Client{
			Client:     retry.DefaultClient,
			Cache:      auth.NewCache(),
			Credential: auth.StaticCredential(repo.Reference.Registry, *credential),
		}
	}

	_, err = oras.Copy(ctx, repo, version, store, version, oras.CopyOptions{
		CopyGraphOptions: oras.CopyGraphOptions{
			PreCopy: store.selector,
		},
	})
	if err != nil {
		log.Error(err, "Unable to copy OCI content to store")

		return err
	}

	return nil
}

// OCIAuthentication returns user supplied credentials from provider variables.
func OCIAuthentication(c configclient.VariablesClient) *auth.Credential {
	username, _ := c.Get(ociUsernameKey)
	password, _ := c.Get(ociPasswordKey)
	accessToken, _ := c.Get(ociAccessTokenKey)
	refreshToken, _ := c.Get(ociRefreshTokenKey)

	if username != "" || password != "" || accessToken != "" || refreshToken != "" {
		return &auth.Credential{
			Username:     username,
			Password:     password,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
	}

	return nil
}

// FetchOCI copies the content of OCI.
func FetchOCI(ctx context.Context, provider operatorv1.GenericProvider, cred *auth.Credential) (mapStore, error) {
	log := log.FromContext(ctx)

	log.Info("Custom fetch configuration OCI url was provided")

	// Prepare components store for the provider type.
	store := NewMapStore(provider)

	err := CopyOCIStore(ctx, provider.GetSpec().FetchConfig.OCI, provider.GetSpec().Version, &store, cred)
	if err != nil {
		log.Error(err, "Unable to copy OCI content")

		return nil, err
	}

	return store, nil
}
