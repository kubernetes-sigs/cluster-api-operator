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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_parseOCISource(t *testing.T) {
	for _, tc := range []struct {
		url           string
		version       string
		expectURL     string
		expectVersion string
	}{
		{
			url:           "registry/image:v1.1.0",
			version:       "v1.2.0",
			expectURL:     "registry/image",
			expectVersion: "v1.1.0",
		},
		{
			url:           "registry/image",
			version:       "v1.2.0",
			expectURL:     "registry/image",
			expectVersion: "v1.2.0",
		},
		{
			url:           "registry:5050/image",
			version:       "v1.2.0",
			expectURL:     "registry:5050/image",
			expectVersion: "v1.2.0",
		},
		{
			url:           "registry:5050/image:v1.1.0",
			version:       "v1.2.0",
			expectURL:     "registry:5050/image",
			expectVersion: "v1.1.0",
		},
		{
			url:           "image",
			version:       "v1.2.0",
			expectURL:     "image",
			expectVersion: "v1.2.0",
		},
	} {
		t.Run(tc.url, func(t *testing.T) {
			g := NewWithT(t)

			{
				url, version, plainHTTP := parseOCISource(tc.url, tc.version)
				g.Expect(url).To(Equal(tc.expectURL))
				g.Expect(version).To(Equal(tc.expectVersion))
				g.Expect(plainHTTP).To(BeFalse())
			}

			{
				url, version, plainHTTP := parseOCISource(fmt.Sprintf("http://%s", tc.url), tc.version)
				g.Expect(url).To(Equal(tc.expectURL))
				g.Expect(version).To(Equal(tc.expectVersion))
				g.Expect(plainHTTP).To(BeTrue())
			}
		})
	}
}
