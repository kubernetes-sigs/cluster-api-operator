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
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	OCIUsernameKey     = "OCI_USERNAME"
	OCIPasswordKey     = "OCI_PASSWORD"
	OCIAccessTokenKey  = "OCI_ACCESS_TOKEN"
	OCIRefreshTokenKey = "OCI_REFRESH_TOKEN" // #nosec G101

	metadataFile     = "metadata.yaml"
	fullMetadataFile = "%s-%s-%s-metadata.yaml"

	componentsFile      = "components.yaml"
	typedComponentsFile = "%s-components.yaml"
	fullComponentsFile  = "%s-%s-%s-components.yaml"
)

// mapStore is a pre-initialized map with expected file names to copy from OCI artifact.
type mapStore struct {
	data   map[string][]byte
	source oras.Target
}

// NewMapStore initializes mapStore for the provider resource.
func NewMapStore(p operatorv1.GenericProvider) mapStore {
	return mapStore{
		data: map[string][]byte{
			metadataFile:   nil,
			componentsFile: nil,
			fmt.Sprintf(typedComponentsFile, p.GetType()):                                       nil,
			fmt.Sprintf(fullMetadataFile, p.GetType(), p.ProviderName(), p.GetSpec().Version):   nil,
			fmt.Sprintf(fullComponentsFile, p.GetType(), p.ProviderName(), p.GetSpec().Version): nil,
		},
	}
}

// GetMetadata returns metadata file for the provider.
func (m mapStore) GetMetadata(p operatorv1.GenericProvider) ([]byte, error) {
	fullMetadataKey := fmt.Sprintf(fullMetadataFile, p.GetType(), p.ProviderName(), p.GetSpec().Version)

	data := m.data[fullMetadataKey]
	if len(data) != 0 {
		return data, nil
	}

	data = m.data[metadataFile]
	if len(data) != 0 {
		return data, nil
	}

	return nil, fmt.Errorf("collected artifact needs to provide metadata as %s or %s file", fullMetadataKey, metadataFile)
}

// GetComponents returns componenents file for the provider.
func (m mapStore) GetComponents(p operatorv1.GenericProvider) ([]byte, error) {
	fullComponentsKey := fmt.Sprintf(fullComponentsFile, p.GetType(), p.ProviderName(), p.GetSpec().Version)

	data := m.data[fullComponentsKey]
	if len(data) != 0 {
		return data, nil
	}

	typedComponentsKey := fmt.Sprintf(typedComponentsFile, p.GetType())

	data = m.data[typedComponentsKey]
	if len(data) != 0 {
		return data, nil
	}

	data = m.data[componentsFile]
	if len(data) != 0 {
		return data, nil
	}

	return nil, fmt.Errorf("collected artifact needs to provide components as %s or %s or %s file", fullComponentsKey, typedComponentsKey, componentsFile)
}

// selector is a PreCopy implementation for the oras.Target which fetches only expected files.
// This helps to reduce the load on the source registry in case required item was added via restoreDuplicates.
func (m mapStore) selector(_ context.Context, desc ocispec.Descriptor) error {
	file := desc.Annotations[ocispec.AnnotationTitle]
	if data := m.data[file]; len(data) == 0 {
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
	return nil, nil //nolint:nilnil
}

// Push implements oras.Target.
func (m mapStore) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) (err error) {
	// Verify we only store expected artifact names
	file := expected.Annotations[ocispec.AnnotationTitle]
	if _, expected := m.data[file]; expected {
		m.data[file], err = io.ReadAll(content)
	}

	if err := m.restoreDuplicates(ctx, expected); err != nil {
		return fmt.Errorf("failed to restore duplicated file: %w", err)
	}

	return err
}

func (m mapStore) restoreDuplicates(ctx context.Context, desc ocispec.Descriptor) (err error) {
	successors, err := content.Successors(ctx, m.source, desc)
	if err != nil {
		return err
	}

	for _, successor := range successors {
		file := successor.Annotations[ocispec.AnnotationTitle]
		if _, expected := m.data[file]; !expected {
			continue
		}

		if err := func() error {
			desc := ocispec.Descriptor{
				MediaType: successor.MediaType,
				Digest:    successor.Digest,
				Size:      successor.Size,
			}
			rc, err := m.source.Fetch(ctx, desc)
			if err != nil {
				return fmt.Errorf("%q: %s: %w", file, desc.MediaType, err)
			}

			defer func() {
				err = rc.Close()
			}()

			if err := m.Push(ctx, successor, rc); err != nil {
				return fmt.Errorf("%q: %s: %w", file, desc.MediaType, err)
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
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

// parseOCISource accepts an OCI URL and the provider version. It returns the image name,
// the image version (if not set on the OCI URL, the provider version is used) and whether
// plain HTTP should be used to fetch the image (when url starts with "http://").
func parseOCISource(url string, version string) (string, string, bool) {
	url, plainHTTP := strings.CutPrefix(url, "http://")

	if parts := strings.SplitN(url, ":", 3); len(parts) == 2 && !strings.Contains(parts[1], "/") {
		url = parts[0]
		version = parts[1]
	} else if len(parts) == 3 {
		version = parts[2]
		url = fmt.Sprintf("%s:%s", parts[0], parts[1])
	}

	return url, version, plainHTTP
}

// CopyOCIStore collects artifacts from the provider OCI url and creates a map of file contents.
func CopyOCIStore(ctx context.Context, url string, version string, store *mapStore, credential *auth.Credential) error {
	log := log.FromContext(ctx)

	url, version, plainHTTP := parseOCISource(url, version)

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

	repo.PlainHTTP = plainHTTP

	// Set the source repository for restoring duplicated content inside the artifact
	store.source = repo

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
	username, _ := c.Get(OCIUsernameKey)
	password, _ := c.Get(OCIPasswordKey)
	accessToken, _ := c.Get(OCIAccessTokenKey)
	refreshToken, _ := c.Get(OCIRefreshTokenKey)

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
func FetchOCI(ctx context.Context, provider operatorv1.GenericProvider, cred *auth.Credential) (*mapStore, error) {
	log := log.FromContext(ctx)

	log.Info("Custom fetch configuration OCI url was provided")

	// Prepare components store for the provider type.
	store := NewMapStore(provider)

	err := CopyOCIStore(ctx, provider.GetSpec().FetchConfig.OCI, provider.GetSpec().Version, &store, cred)
	if err != nil {
		log.Error(err, "Unable to copy OCI content")

		return nil, err
	}

	return &store, nil
}
