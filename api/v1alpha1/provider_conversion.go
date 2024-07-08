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

	apimachineryconversion "k8s.io/apimachinery/pkg/conversion"
	"k8s.io/utils/ptr"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	ctrlconfigv1 "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
)

// ConvertTo converts this BootstrapProvider to the Hub version (v1alpha2).
func (src *BootstrapProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.BootstrapProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.BootstrapProvider")
	}

	if err := Convert_v1alpha1_BootstrapProvider_To_v1alpha2_BootstrapProvider(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &operatorv1.BootstrapProvider{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	dst.Spec.ManifestPatches = restored.Spec.ManifestPatches
	dst.Spec.AdditionalDeployments = restored.Spec.AdditionalDeployments

	return nil
}

// ConvertFrom converts from the BootstrapProvider version (v1alpha2) to this version.
func (dst *BootstrapProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.BootstrapProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.BootstrapProvider")
	}

	if err := Convert_v1alpha2_BootstrapProvider_To_v1alpha1_BootstrapProvider(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion.
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this BootstrapProviderList to the Hub version (v1alpha2).
func (src *BootstrapProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.BootstrapProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.BootstrapProviderList")
	}

	return Convert_v1alpha1_BootstrapProviderList_To_v1alpha2_BootstrapProviderList(src, dst, nil)
}

// ConvertFrom converts from the BootstrapProviderList version (v1alpha2) to this version.
func (dst *BootstrapProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.BootstrapProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.BootstrapProviderList")
	}

	return Convert_v1alpha2_BootstrapProviderList_To_v1alpha1_BootstrapProviderList(src, dst, nil)
}

// ConvertTo converts this ControlPlaneProvider to the Hub version (v1alpha2).
func (src *ControlPlaneProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.ControlPlaneProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.ControlPlaneProvider")
	}

	if err := Convert_v1alpha1_ControlPlaneProvider_To_v1alpha2_ControlPlaneProvider(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &operatorv1.ControlPlaneProvider{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	dst.Spec.ManifestPatches = restored.Spec.ManifestPatches
	dst.Spec.AdditionalDeployments = restored.Spec.AdditionalDeployments

	return nil
}

// ConvertFrom converts from the ControlPlaneProvider version (v1alpha2) to this version.
func (dst *ControlPlaneProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.ControlPlaneProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.ControlPlaneProvider")
	}

	if err := Convert_v1alpha2_ControlPlaneProvider_To_v1alpha1_ControlPlaneProvider(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion.
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this ControlPlaneProviderList to the Hub version (v1alpha2).
func (src *ControlPlaneProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.ControlPlaneProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.ControlPlaneProviderList")
	}

	return Convert_v1alpha1_ControlPlaneProviderList_To_v1alpha2_ControlPlaneProviderList(src, dst, nil)
}

// ConvertFrom converts from the ControlPlaneProviderList version (v1alpha2) to this version.
func (dst *ControlPlaneProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.ControlPlaneProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.ControlPlaneProviderList")
	}

	return Convert_v1alpha2_ControlPlaneProviderList_To_v1alpha1_ControlPlaneProviderList(src, dst, nil)
}

// ConvertTo converts this CoreProvider to the Hub version (v1alpha2).
func (src *CoreProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.CoreProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.CoreProvider")
	}

	if err := Convert_v1alpha1_CoreProvider_To_v1alpha2_CoreProvider(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &operatorv1.CoreProvider{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	dst.Spec.ManifestPatches = restored.Spec.ManifestPatches
	dst.Spec.AdditionalDeployments = restored.Spec.AdditionalDeployments

	return nil
}

// ConvertFrom converts from the CoreProvider version (v1alpha2) to this version.
func (dst *CoreProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.CoreProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.CoreProvider")
	}

	if err := Convert_v1alpha2_CoreProvider_To_v1alpha1_CoreProvider(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion.
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this CoreProviderList to the Hub version (v1alpha2).
func (src *CoreProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.CoreProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.CoreProviderList")
	}

	return Convert_v1alpha1_CoreProviderList_To_v1alpha2_CoreProviderList(src, dst, nil)
}

// ConvertFrom converts from the CoreProviderList version (v1alpha2) to this version.
func (dst *CoreProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.CoreProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.CoreProviderList")
	}

	return Convert_v1alpha2_CoreProviderList_To_v1alpha1_CoreProviderList(src, dst, nil)
}

// ConvertTo converts this InfrastructureProvider to the Hub version (v1alpha2).
func (src *InfrastructureProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.InfrastructureProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.InfrastructureProvider")
	}

	if err := Convert_v1alpha1_InfrastructureProvider_To_v1alpha2_InfrastructureProvider(src, dst, nil); err != nil {
		return err
	}

	// Manually restore data.
	restored := &operatorv1.InfrastructureProvider{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}

	dst.Spec.ManifestPatches = restored.Spec.ManifestPatches
	dst.Spec.AdditionalDeployments = restored.Spec.AdditionalDeployments

	return nil
}

// ConvertFrom converts from the InfrastructureProvider version (v1alpha2) to this version.
func (dst *InfrastructureProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.InfrastructureProvider)
	if !ok {
		panic("expected to get an of object of type v1alpha2.InfrastructureProvider")
	}

	if err := Convert_v1alpha2_InfrastructureProvider_To_v1alpha1_InfrastructureProvider(src, dst, nil); err != nil {
		return err
	}

	// Preserve Hub data on down-conversion.
	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this InfrastructureProviderList to the Hub version (v1alpha2).
func (src *InfrastructureProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.InfrastructureProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.InfrastructureProviderList")
	}

	return Convert_v1alpha1_InfrastructureProviderList_To_v1alpha2_InfrastructureProviderList(src, dst, nil)
}

// ConvertFrom converts from the InfrastructureProviderList version (v1alpha2) to this version.
func (dst *InfrastructureProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.InfrastructureProviderList)
	if !ok {
		panic("expected to get an of object of type v1alpha2.InfrastructureProviderList")
	}

	return Convert_v1alpha2_InfrastructureProviderList_To_v1alpha1_InfrastructureProviderList(src, dst, nil)
}

func Convert_v1alpha1_ManagerSpec_To_v1alpha2_ManagerSpec(in *ManagerSpec, out *operatorv1.ManagerSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.ControllerManagerConfiguration.SyncPeriod = in.ControllerManagerConfigurationSpec.SyncPeriod
	out.ControllerManagerConfiguration.LeaderElection = in.ControllerManagerConfigurationSpec.LeaderElection
	out.ControllerManagerConfiguration.CacheNamespace = in.ControllerManagerConfigurationSpec.CacheNamespace
	out.ControllerManagerConfiguration.GracefulShutdownTimeout = in.ControllerManagerConfigurationSpec.GracefulShutdownTimeout

	if in.ControllerManagerConfigurationSpec.Controller != nil {
		out.ControllerManagerConfiguration.Controller = &operatorv1.ControllerConfigurationSpec{
			GroupKindConcurrency: in.ControllerManagerConfigurationSpec.Controller.GroupKindConcurrency,
			CacheSyncTimeout:     in.ControllerManagerConfigurationSpec.Controller.CacheSyncTimeout,
			RecoverPanic:         in.ControllerManagerConfigurationSpec.Controller.RecoverPanic,
		}
	}

	out.ControllerManagerConfiguration.Metrics = operatorv1.ControllerMetrics{
		BindAddress: in.ControllerManagerConfigurationSpec.Metrics.BindAddress,
	}

	out.ControllerManagerConfiguration.Health = operatorv1.ControllerHealth{
		HealthProbeBindAddress: in.ControllerManagerConfigurationSpec.Health.HealthProbeBindAddress,
		ReadinessEndpointName:  in.ControllerManagerConfigurationSpec.Health.ReadinessEndpointName,
		LivenessEndpointName:   in.ControllerManagerConfigurationSpec.Health.LivenessEndpointName,
	}

	out.ControllerManagerConfiguration.Webhook = operatorv1.ControllerWebhook{
		Port:    in.ControllerManagerConfigurationSpec.Webhook.Port,
		Host:    in.ControllerManagerConfigurationSpec.Webhook.Host,
		CertDir: in.ControllerManagerConfigurationSpec.Webhook.CertDir,
	}

	out.ProfilerAddress = in.ProfilerAddress
	out.MaxConcurrentReconciles = in.MaxConcurrentReconciles
	out.Verbosity = in.Verbosity
	out.FeatureGates = in.FeatureGates

	return nil
}

func Convert_v1alpha2_ManagerSpec_To_v1alpha1_ManagerSpec(in *operatorv1.ManagerSpec, out *ManagerSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.ControllerManagerConfigurationSpec.SyncPeriod = in.ControllerManagerConfiguration.SyncPeriod
	out.ControllerManagerConfigurationSpec.LeaderElection = in.ControllerManagerConfiguration.LeaderElection
	out.ControllerManagerConfigurationSpec.CacheNamespace = in.ControllerManagerConfiguration.CacheNamespace
	out.ControllerManagerConfigurationSpec.GracefulShutdownTimeout = in.ControllerManagerConfiguration.GracefulShutdownTimeout

	if in.ControllerManagerConfiguration.Controller != nil {
		out.ControllerManagerConfigurationSpec.Controller = &ctrlconfigv1.ControllerConfigurationSpec{
			GroupKindConcurrency: in.ControllerManagerConfiguration.Controller.GroupKindConcurrency,
			CacheSyncTimeout:     in.ControllerManagerConfiguration.Controller.CacheSyncTimeout,
			RecoverPanic:         in.ControllerManagerConfiguration.Controller.RecoverPanic,
		}
	}

	out.ControllerManagerConfigurationSpec.Metrics = ctrlconfigv1.ControllerMetrics{
		BindAddress: in.ControllerManagerConfiguration.Metrics.BindAddress,
	}

	out.ControllerManagerConfigurationSpec.Health = ctrlconfigv1.ControllerHealth{
		HealthProbeBindAddress: in.ControllerManagerConfiguration.Health.HealthProbeBindAddress,
		ReadinessEndpointName:  in.ControllerManagerConfiguration.Health.ReadinessEndpointName,
		LivenessEndpointName:   in.ControllerManagerConfiguration.Health.LivenessEndpointName,
	}

	out.ControllerManagerConfigurationSpec.Webhook = ctrlconfigv1.ControllerWebhook{
		Port:    in.ControllerManagerConfiguration.Webhook.Port,
		Host:    in.ControllerManagerConfiguration.Webhook.Host,
		CertDir: in.ControllerManagerConfiguration.Webhook.CertDir,
	}

	out.ProfilerAddress = in.ProfilerAddress
	out.MaxConcurrentReconciles = in.MaxConcurrentReconciles
	out.Verbosity = in.Verbosity
	out.FeatureGates = in.FeatureGates

	return nil
}

func Convert_v1alpha1_ProviderSpec_To_v1alpha2_ProviderSpec(in *ProviderSpec, out *operatorv1.ProviderSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.Version = in.Version

	if in.Manager != nil {
		out.Manager = &operatorv1.ManagerSpec{}

		if err := Convert_v1alpha1_ManagerSpec_To_v1alpha2_ManagerSpec(in.Manager, out.Manager, nil); err != nil {
			return err
		}
	}

	if in.Deployment != nil {
		out.Deployment = &operatorv1.DeploymentSpec{}

		if err := Convert_v1alpha1_DeploymentSpec_To_v1alpha2_DeploymentSpec(in.Deployment, out.Deployment, nil); err != nil {
			return err
		}
	}

	if in.FetchConfig != nil {
		out.FetchConfig = &operatorv1.FetchConfiguration{}

		if err := Convert_v1alpha1_FetchConfiguration_To_v1alpha2_FetchConfiguration(in.FetchConfig, out.FetchConfig, nil); err != nil {
			return err
		}
	}

	if in.SecretName != "" || in.SecretNamespace != "" {
		out.ConfigSecret = &operatorv1.SecretReference{
			Name:      in.SecretName,
			Namespace: in.SecretNamespace,
		}
	}

	if in.AdditionalManifestsRef != nil {
		out.AdditionalManifestsRef = &operatorv1.ConfigmapReference{
			Name:      in.AdditionalManifestsRef.Name,
			Namespace: in.AdditionalManifestsRef.Namespace,
		}
	}

	return nil
}

func Convert_v1alpha2_ProviderSpec_To_v1alpha1_ProviderSpec(in *operatorv1.ProviderSpec, out *ProviderSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.Version = in.Version

	if in.Manager != nil {
		out.Manager = &ManagerSpec{}

		if err := Convert_v1alpha2_ManagerSpec_To_v1alpha1_ManagerSpec(in.Manager, out.Manager, nil); err != nil {
			return err
		}
	}

	if in.Deployment != nil {
		out.Deployment = &DeploymentSpec{}

		if err := Convert_v1alpha2_DeploymentSpec_To_v1alpha1_DeploymentSpec(in.Deployment, out.Deployment, nil); err != nil {
			return err
		}
	}

	if in.FetchConfig != nil {
		out.FetchConfig = &FetchConfiguration{}

		if err := Convert_v1alpha2_FetchConfiguration_To_v1alpha1_FetchConfiguration(in.FetchConfig, out.FetchConfig, nil); err != nil {
			return err
		}
	}

	if in.ConfigSecret != nil {
		out.SecretName = in.ConfigSecret.Name
		out.SecretNamespace = in.ConfigSecret.Namespace
	}

	if in.AdditionalManifestsRef != nil {
		out.AdditionalManifestsRef = &ConfigmapReference{
			Name:      in.AdditionalManifestsRef.Name,
			Namespace: in.AdditionalManifestsRef.Namespace,
		}
	}

	return nil
}

func Convert_v1alpha1_ContainerSpec_To_v1alpha2_ContainerSpec(in *ContainerSpec, out *operatorv1.ContainerSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.Name = in.Name

	if in.Image != nil {
		out.ImageURL = fromImageMeta(in.Image)
	} else {
		out.ImageURL = nil
	}

	out.Args = in.Args
	out.Env = in.Env
	out.Resources = in.Resources
	out.Command = in.Command

	return nil
}

func Convert_v1alpha2_ContainerSpec_To_v1alpha1_ContainerSpec(in *operatorv1.ContainerSpec, out *ContainerSpec, s apimachineryconversion.Scope) error {
	if in == nil {
		return nil
	}

	out.Name = in.Name

	if in.ImageURL != nil {
		out.Image = toImageMeta(*in.ImageURL)
	} else {
		out.Image = nil
	}

	out.Args = in.Args
	out.Env = in.Env
	out.Resources = in.Resources
	out.Command = in.Command

	return nil
}

func toImageMeta(imageURL string) *ImageMeta {
	im := ImageMeta{}

	// Split the URL by the last "/"" to separate the registry address
	lastInd := strings.LastIndex(imageURL, "/")

	if lastInd != -1 {
		im.Repository = imageURL[:lastInd]

		imageURL = imageURL[lastInd+1:]
	}

	// Now split by ":" to separate the tag
	urlSplit := strings.Split(imageURL, ":")
	if len(urlSplit) == 2 {
		im.Tag = urlSplit[1]
	}

	im.Name = urlSplit[0]

	return &im
}

func fromImageMeta(im *ImageMeta) *string {
	result := strings.Join([]string{im.Repository, im.Name}, "/")
	if im.Repository == "" {
		result = im.Name
	}

	if im.Tag != "" {
		result = result + ":" + im.Tag
	}

	return ptr.To(result)
}
