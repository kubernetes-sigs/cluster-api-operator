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

package v1alpha3

// Hub marks v1alpha3 as the hub version for conversions.
// This means v1alpha3 is the canonical internal representation,
// and all other versions convert to/from this version.
func (*CoreProvider) Hub()                 {}
func (*CoreProviderList) Hub()             {}
func (*BootstrapProvider) Hub()            {}
func (*BootstrapProviderList) Hub()        {}
func (*ControlPlaneProvider) Hub()         {}
func (*ControlPlaneProviderList) Hub()     {}
func (*InfrastructureProvider) Hub()       {}
func (*InfrastructureProviderList) Hub()   {}
func (*AddonProvider) Hub()                {}
func (*AddonProviderList) Hub()            {}
func (*IPAMProvider) Hub()                 {}
func (*IPAMProviderList) Hub()             {}
func (*RuntimeExtensionProvider) Hub()     {}
func (*RuntimeExtensionProviderList) Hub() {}
