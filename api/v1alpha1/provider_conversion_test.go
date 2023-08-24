/*
Copyright 2023 The Kubernetes Authors.

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

package v1alpha1

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	fuzz "github.com/google/gofuzz"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
)

func TestFuzzyConversion(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(Succeed())
	g.Expect(operatorv1.AddToScheme(scheme)).To(Succeed())

	t.Run("for CoreProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &operatorv1.CoreProvider{},
		Spoke:       &CoreProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{imageMetaFuzzFunc, imageURLFuzzFunc, secretConfigFuzzFunc},
	}))

	t.Run("for ControlPlaneProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &operatorv1.ControlPlaneProvider{},
		Spoke:       &ControlPlaneProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{imageMetaFuzzFunc, imageURLFuzzFunc, secretConfigFuzzFunc},
	}))

	t.Run("for BootstrapProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &operatorv1.BootstrapProvider{},
		Spoke:       &BootstrapProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{imageMetaFuzzFunc, imageURLFuzzFunc, secretConfigFuzzFunc},
	}))

	t.Run("for InfrastructureProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &operatorv1.InfrastructureProvider{},
		Spoke:       &InfrastructureProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{imageMetaFuzzFunc, imageURLFuzzFunc, secretConfigFuzzFunc},
	}))
}

func secretConfigFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		secretConfigFuzzer,
	}
}

func secretConfigFuzzer(in *operatorv1.SecretReference, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// if ConfigSecret is set, its name cannot be nil.
	if in != nil && in.Name == "" {
		in.Name = c.RandString() + "name"
	}
}

func imageURLFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		imageURLFuzzer,
	}
}

func imageURLFuzzer(in *operatorv1.ContainerSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// There is a separate test for image URL conversion.
	in.ImageURL = nil
}

func imageMetaFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		imageMetaFuzzer,
	}
}

func imageMetaFuzzer(in *ImageMeta, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// Name, Repository and Tag cannot contain '/' and ':' characters
	in.Name = replaceSpecialCharacters(in.Name, "/", ":")
	in.Repository = replaceSpecialCharacters(in.Repository, "/", ":")
	in.Tag = replaceSpecialCharacters(in.Tag, "/", ":")
}

func replaceSpecialCharacters(str string, characters ...string) string {
	for _, c := range characters {
		str = strings.ReplaceAll(str, c, "")
	}

	return str
}

func TestConvertImageMeta(t *testing.T) {
	testCases := []struct {
		name     string
		imageURL string
	}{
		{
			name:     "empty url",
			imageURL: "",
		},
		{
			name:     "full url",
			imageURL: "registry/namespace/image:tag",
		},
		{
			name:     "no registry",
			imageURL: "image:tag",
		},
		{
			name:     "no tag",
			imageURL: "registry/namespace/image",
		},
		{
			name:     "no namespace",
			imageURL: "registry/image:tag",
		},
		{
			name:     "only name",
			imageURL: "image",
		},
		{
			name:     "with port",
			imageURL: "registry:5000/namespace/image:tag",
		},
		{
			name:     "with port, without tag",
			imageURL: "registry:5000/namespace/image",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			convertedURL := fromImageMeta(toImageMeta(tc.imageURL))

			g.Expect(tc.imageURL).To(Equal(*convertedURL))
		})
	}
}
