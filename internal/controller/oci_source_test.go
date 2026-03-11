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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// generateTestCA creates a self-signed CA certificate and returns the PEM-encoded cert bytes.
func generateTestCA(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func Test_transportWithCA(t *testing.T) {
	caPEM := generateTestCA(t)

	t.Run("valid CA cert", func(t *testing.T) {
		g := NewWithT(t)

		transport, err := transportWithCA(caPEM)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(transport).ToNot(BeNil())

		httpTransport, ok := transport.(*http.Transport)
		g.Expect(ok).To(BeTrue())
		g.Expect(httpTransport.TLSClientConfig).ToNot(BeNil())
		g.Expect(httpTransport.TLSClientConfig.RootCAs).ToNot(BeNil())
		g.Expect(httpTransport.TLSClientConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
	})

	t.Run("invalid PEM data", func(t *testing.T) {
		g := NewWithT(t)

		transport, err := transportWithCA([]byte("not-a-valid-pem"))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("failed to append custom CA certificate"))
		g.Expect(transport).To(BeNil())
	})
}

func Test_newRetryClient(t *testing.T) {
	t.Run("nil base transport uses retry default", func(t *testing.T) {
		g := NewWithT(t)

		client := newRetryClient(nil)
		g.Expect(client).ToNot(BeNil())
		g.Expect(client.Transport).ToNot(BeNil())

		retryTransport, ok := client.Transport.(*retry.Transport)
		g.Expect(ok).To(BeTrue())
		g.Expect(retryTransport.Base).To(BeNil())
	})

	t.Run("custom base transport is wrapped in retry", func(t *testing.T) {
		g := NewWithT(t)

		caPEM := generateTestCA(t)
		base, err := transportWithCA(caPEM)
		g.Expect(err).ToNot(HaveOccurred())

		client := newRetryClient(base)
		g.Expect(client).ToNot(BeNil())

		retryTransport, ok := client.Transport.(*retry.Transport)
		g.Expect(ok).To(BeTrue())
		g.Expect(retryTransport.Base).To(Equal(base))
	})

	t.Run("does not mutate retry.DefaultClient", func(t *testing.T) {
		g := NewWithT(t)

		originalTransport := retry.DefaultClient.Transport

		caPEM := generateTestCA(t)
		base, err := transportWithCA(caPEM)
		g.Expect(err).ToNot(HaveOccurred())

		_ = newRetryClient(base)

		g.Expect(retry.DefaultClient.Transport).To(BeIdenticalTo(originalTransport))
	})
}

func Test_CopyOCIStore_clientSetup(t *testing.T) {
	caPEM := generateTestCA(t)

	t.Run("with caCert and credentials, auth client wraps custom transport", func(t *testing.T) {
		g := NewWithT(t)

		// We can't fully test CopyOCIStore without a real registry, but we can
		// verify the client setup by checking the repo's client after construction.
		// Use the helper functions directly to verify the composition.
		base, err := transportWithCA(caPEM)
		g.Expect(err).ToNot(HaveOccurred())

		baseClient := newRetryClient(base)

		cred := &auth.Credential{Username: "user", Password: "pass"}
		authClient := &auth.Client{
			Client:     baseClient,
			Cache:      auth.NewCache(),
			Credential: auth.StaticCredential("registry.example.com", *cred),
		}

		// The auth client's inner client should be our custom baseClient
		g.Expect(authClient.Client).To(BeIdenticalTo(baseClient))

		// The baseClient's transport should wrap our custom CA transport
		retryTransport, ok := baseClient.Transport.(*retry.Transport)
		g.Expect(ok).To(BeTrue())
		g.Expect(retryTransport.Base).To(Equal(base))
	})

	t.Run("without caCert, default transport is used", func(t *testing.T) {
		g := NewWithT(t)

		baseClient := newRetryClient(nil)

		retryTransport, ok := baseClient.Transport.(*retry.Transport)
		g.Expect(ok).To(BeTrue())
		g.Expect(retryTransport.Base).To(BeNil())
	})
}
