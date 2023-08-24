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

package v1alpha2

// Hub marks this API version as a conversion hub for BootstrapProvider.
func (*BootstrapProvider) Hub() {}

// Hub marks this API version as a conversion hub for BootstrapProviderList.
func (*BootstrapProviderList) Hub() {}

// Hub marks this API version as a conversion hub for ControlPlaneProvider.
func (*ControlPlaneProvider) Hub() {}

// Hub marks this API version as a conversion hub for ControlPlaneProviderList.
func (*ControlPlaneProviderList) Hub() {}

// Hub marks this API version as a conversion hub for CoreProvider.
func (*CoreProvider) Hub() {}

// Hub marks this API version as a conversion hub for CoreProviderList.
func (*CoreProviderList) Hub() {}

// Hub marks this API version as a conversion hub for InfrastructureProvider.
func (*InfrastructureProvider) Hub() {}

// Hub marks this API version as a conversion hub for InfrastructureProviderList.
func (*InfrastructureProviderList) Hub() {}
