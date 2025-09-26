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

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

// ConvertTo converts this CoreProvider to the Hub version (v1alpha3).
func (src *CoreProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.CoreProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.CoreProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_CoreProvider_To_v1alpha3_CoreProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *CoreProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.CoreProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.CoreProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_CoreProvider_To_v1alpha2_CoreProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this CoreProviderList to the Hub version (v1alpha3).
func (src *CoreProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.CoreProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.CoreProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_CoreProviderList_To_v1alpha3_CoreProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *CoreProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.CoreProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.CoreProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_CoreProviderList_To_v1alpha2_CoreProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// Similar conversion methods for other provider types...
// BootstrapProvider conversions.

// ConvertTo converts this BootstrapProvider to the Hub version (v1alpha3).
func (src *BootstrapProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.BootstrapProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.BootstrapProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_BootstrapProvider_To_v1alpha3_BootstrapProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *BootstrapProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.BootstrapProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.BootstrapProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_BootstrapProvider_To_v1alpha2_BootstrapProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this BootstrapProviderList to the Hub version (v1alpha3).
func (src *BootstrapProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.BootstrapProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.BootstrapProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_BootstrapProviderList_To_v1alpha3_BootstrapProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *BootstrapProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.BootstrapProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.BootstrapProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_BootstrapProviderList_To_v1alpha2_BootstrapProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ControlPlaneProvider conversions.

// ConvertTo converts this ControlPlaneProvider to the Hub version (v1alpha3).
func (src *ControlPlaneProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.ControlPlaneProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.ControlPlaneProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_ControlPlaneProvider_To_v1alpha3_ControlPlaneProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *ControlPlaneProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.ControlPlaneProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.ControlPlaneProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_ControlPlaneProvider_To_v1alpha2_ControlPlaneProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this ControlPlaneProviderList to the Hub version (v1alpha3).
func (src *ControlPlaneProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.ControlPlaneProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.ControlPlaneProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_ControlPlaneProviderList_To_v1alpha3_ControlPlaneProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *ControlPlaneProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.ControlPlaneProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.ControlPlaneProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_ControlPlaneProviderList_To_v1alpha2_ControlPlaneProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// InfrastructureProvider conversions.

// ConvertTo converts this InfrastructureProvider to the Hub version (v1alpha3).
func (src *InfrastructureProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.InfrastructureProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.InfrastructureProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_InfrastructureProvider_To_v1alpha3_InfrastructureProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *InfrastructureProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.InfrastructureProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.InfrastructureProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_InfrastructureProvider_To_v1alpha2_InfrastructureProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this InfrastructureProviderList to the Hub version (v1alpha3).
func (src *InfrastructureProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.InfrastructureProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.InfrastructureProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_InfrastructureProviderList_To_v1alpha3_InfrastructureProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *InfrastructureProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.InfrastructureProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.InfrastructureProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_InfrastructureProviderList_To_v1alpha2_InfrastructureProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// AddonProvider conversions.

// ConvertTo converts this AddonProvider to the Hub version (v1alpha3).
func (src *AddonProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.AddonProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.AddonProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_AddonProvider_To_v1alpha3_AddonProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *AddonProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.AddonProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.AddonProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_AddonProvider_To_v1alpha2_AddonProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this AddonProviderList to the Hub version (v1alpha3).
func (src *AddonProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.AddonProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.AddonProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_AddonProviderList_To_v1alpha3_AddonProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *AddonProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.AddonProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.AddonProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_AddonProviderList_To_v1alpha2_AddonProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// IPAMProvider conversions.

// ConvertTo converts this IPAMProvider to the Hub version (v1alpha3).
func (src *IPAMProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.IPAMProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.IPAMProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_IPAMProvider_To_v1alpha3_IPAMProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *IPAMProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.IPAMProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.IPAMProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_IPAMProvider_To_v1alpha2_IPAMProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this IPAMProviderList to the Hub version (v1alpha3).
func (src *IPAMProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.IPAMProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.IPAMProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_IPAMProviderList_To_v1alpha3_IPAMProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *IPAMProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.IPAMProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.IPAMProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_IPAMProviderList_To_v1alpha2_IPAMProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// RuntimeExtensionProvider conversions.

// ConvertTo converts this RuntimeExtensionProvider to the Hub version (v1alpha3).
func (src *RuntimeExtensionProvider) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.RuntimeExtensionProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.RuntimeExtensionProvider, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_RuntimeExtensionProvider_To_v1alpha3_RuntimeExtensionProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *RuntimeExtensionProvider) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.RuntimeExtensionProvider)
	if !ok {
		return fmt.Errorf("expected v1alpha3.RuntimeExtensionProvider, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_RuntimeExtensionProvider_To_v1alpha2_RuntimeExtensionProvider(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertTo converts this RuntimeExtensionProviderList to the Hub version (v1alpha3).
func (src *RuntimeExtensionProviderList) ConvertTo(dstRaw conversion.Hub) error {
	dst, ok := dstRaw.(*operatorv1.RuntimeExtensionProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.RuntimeExtensionProviderList, got %T", dstRaw)
	}

	if err := Convert_v1alpha2_RuntimeExtensionProviderList_To_v1alpha3_RuntimeExtensionProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1alpha3) to this version.
func (dst *RuntimeExtensionProviderList) ConvertFrom(srcRaw conversion.Hub) error {
	src, ok := srcRaw.(*operatorv1.RuntimeExtensionProviderList)
	if !ok {
		return fmt.Errorf("expected v1alpha3.RuntimeExtensionProviderList, got %T", srcRaw)
	}

	if err := Convert_v1alpha3_RuntimeExtensionProviderList_To_v1alpha2_RuntimeExtensionProviderList(src, dst); err != nil {
		return err
	}

	return nil
}

// Helper function to convert ManagerSpec to Args in ContainerSpec.
func convertManagerSpecToArgs(mSpec *ManagerSpec) map[string]string {
	if mSpec == nil {
		return nil
	}

	args := make(map[string]string)

	// ControllerManagerConfigurationSpec fields
	if mSpec.Controller != nil {
		for k, v := range mSpec.Controller.GroupKindConcurrency {
			args["--"+strings.ToLower(k)+"-concurrency"] = strconv.Itoa(v)
		}
		// Note: CacheSyncTimeout and RecoverPanic don't have direct arg mappings.
	}

	if mSpec.MaxConcurrentReconciles != 0 {
		args["--max-concurrent-reconciles"] = strconv.Itoa(mSpec.MaxConcurrentReconciles)
	}

	if mSpec.CacheNamespace != "" {
		args["--namespace"] = mSpec.CacheNamespace
	}

	if mSpec.Health.HealthProbeBindAddress != "" {
		args["--health-addr"] = mSpec.Health.HealthProbeBindAddress
	}
	// Note: LivenessEndpointName and ReadinessEndpointName are handled differently (probe paths)

	if mSpec.LeaderElection != nil && mSpec.LeaderElection.LeaderElect != nil {
		args["--leader-elect"] = strconv.FormatBool(*mSpec.LeaderElection.LeaderElect)

		if *mSpec.LeaderElection.LeaderElect {
			if mSpec.LeaderElection.ResourceName != "" && mSpec.LeaderElection.ResourceNamespace != "" {
				args["--leader-election-id"] = mSpec.LeaderElection.ResourceNamespace + "/" + mSpec.LeaderElection.ResourceName
			}

			if mSpec.LeaderElection.LeaseDuration.Duration > 0 {
				leaseDuration := int(mSpec.LeaderElection.LeaseDuration.Duration.Round(time.Second).Seconds())
				args["--leader-elect-lease-duration"] = fmt.Sprintf("%ds", leaseDuration)
			}

			if mSpec.LeaderElection.RenewDeadline.Duration > 0 {
				renewDuration := int(mSpec.LeaderElection.RenewDeadline.Duration.Round(time.Second).Seconds())
				args["--leader-elect-renew-deadline"] = fmt.Sprintf("%ds", renewDuration)
			}

			if mSpec.LeaderElection.RetryPeriod.Duration > 0 {
				retryDuration := int(mSpec.LeaderElection.RetryPeriod.Duration.Round(time.Second).Seconds())
				args["--leader-elect-retry-period"] = fmt.Sprintf("%ds", retryDuration)
			}
		}
	}

	if mSpec.Metrics.BindAddress != "" {
		args["--metrics-bind-addr"] = mSpec.Metrics.BindAddress
	}

	// webhooks
	if mSpec.Webhook.Host != "" {
		args["--webhook-host"] = mSpec.Webhook.Host
	}

	if mSpec.Webhook.Port != nil {
		args["--webhook-port"] = strconv.Itoa(*mSpec.Webhook.Port)
	}

	if mSpec.Webhook.CertDir != "" {
		args["--webhook-cert-dir"] = mSpec.Webhook.CertDir
	}

	// top level fields
	if mSpec.SyncPeriod != nil {
		syncPeriod := int(mSpec.SyncPeriod.Duration.Round(time.Second).Seconds())
		if syncPeriod > 0 {
			args["--sync-period"] = fmt.Sprintf("%ds", syncPeriod)
		}
	}

	if mSpec.ProfilerAddress != "" {
		args["--profiler-address"] = mSpec.ProfilerAddress
	}

	if mSpec.Verbosity != 0 && mSpec.Verbosity != 1 { // 1 is the default
		args["--v"] = strconv.Itoa(mSpec.Verbosity)
	}

	if len(mSpec.FeatureGates) > 0 {
		var fgValues []string
		for fg, val := range mSpec.FeatureGates {
			fgValues = append(fgValues, fmt.Sprintf("%s=%t", fg, val))
		}

		sort.Strings(fgValues)
		args["--feature-gates"] = strings.Join(fgValues, ",")
	}

	// Merge additional args
	for k, v := range mSpec.AdditionalArgs {
		args[k] = v
	}

	return args
}

// Helper function to convert Args back to ManagerSpec.
func convertArgsToManagerSpec(args map[string]string) *ManagerSpec {
	if len(args) == 0 {
		return nil
	}

	mSpec := &ManagerSpec{
		ControllerManagerConfiguration: ControllerManagerConfiguration{
			Controller: &ControllerConfigurationSpec{
				GroupKindConcurrency: make(map[string]int),
			},
			Health:  ControllerHealth{},
			Metrics: ControllerMetrics{},
			Webhook: ControllerWebhook{},
		},
		FeatureGates: make(map[string]bool),
	}

	// Track which args we've processed
	processedArgs := make(map[string]bool)

	for k, v := range args {
		switch {
		// Handle concurrency args
		case strings.HasSuffix(k, "-concurrency"):
			resourceName := strings.TrimSuffix(strings.TrimPrefix(k, "--"), "-concurrency")

			if val, err := strconv.Atoi(v); err == nil {
				mSpec.Controller.GroupKindConcurrency[resourceName] = val
				processedArgs[k] = true
			}

		case k == "--max-concurrent-reconciles":
			if val, err := strconv.Atoi(v); err == nil {
				mSpec.MaxConcurrentReconciles = val
				processedArgs[k] = true
			}

		case k == "--namespace":
			mSpec.CacheNamespace = v
			processedArgs[k] = true

		case k == "--health-addr":
			mSpec.Health.HealthProbeBindAddress = v
			processedArgs[k] = true

		case k == "--leader-elect":
			if val, err := strconv.ParseBool(v); err == nil {
				mSpec.LeaderElection = &configv1alpha1.LeaderElectionConfiguration{
					LeaderElect: &val,
				}
				processedArgs[k] = true
			}

		case k == "--leader-election-id":
			if mSpec.LeaderElection == nil {
				mSpec.LeaderElection = &configv1alpha1.LeaderElectionConfiguration{}
			}

			parts := strings.SplitN(v, "/", 2)
			if len(parts) == 2 {
				mSpec.LeaderElection.ResourceNamespace = parts[0]
				mSpec.LeaderElection.ResourceName = parts[1]
			}

			processedArgs[k] = true

		case k == "--leader-elect-lease-duration":
			if mSpec.LeaderElection == nil {
				mSpec.LeaderElection = &configv1alpha1.LeaderElectionConfiguration{}
			}

			if duration, err := parseDurationString(v); err == nil {
				mSpec.LeaderElection.LeaseDuration = metav1.Duration{Duration: duration}
				processedArgs[k] = true
			}

		case k == "--leader-elect-renew-deadline":
			if mSpec.LeaderElection == nil {
				mSpec.LeaderElection = &configv1alpha1.LeaderElectionConfiguration{}
			}

			if duration, err := parseDurationString(v); err == nil {
				mSpec.LeaderElection.RenewDeadline = metav1.Duration{Duration: duration}
				processedArgs[k] = true
			}

		case k == "--leader-elect-retry-period":
			if mSpec.LeaderElection == nil {
				mSpec.LeaderElection = &configv1alpha1.LeaderElectionConfiguration{}
			}

			if duration, err := parseDurationString(v); err == nil {
				mSpec.LeaderElection.RetryPeriod = metav1.Duration{Duration: duration}
				processedArgs[k] = true
			}

		case k == "--metrics-bind-addr":
			mSpec.Metrics.BindAddress = v
			processedArgs[k] = true

		case k == "--webhook-host":
			mSpec.Webhook.Host = v
			processedArgs[k] = true

		case k == "--webhook-port":
			if val, err := strconv.Atoi(v); err == nil {
				mSpec.Webhook.Port = &val
				processedArgs[k] = true
			}

		case k == "--webhook-cert-dir":
			mSpec.Webhook.CertDir = v
			processedArgs[k] = true

		case k == "--sync-period":
			if duration, err := parseDurationString(v); err == nil {
				mSpec.SyncPeriod = &metav1.Duration{Duration: duration}
				processedArgs[k] = true
			}

		case k == "--profiler-address":
			mSpec.ProfilerAddress = v
			processedArgs[k] = true

		case k == "--v":
			if val, err := strconv.Atoi(v); err == nil {
				mSpec.Verbosity = val
				processedArgs[k] = true
			}

		case k == "--feature-gates":
			features := strings.Split(v, ",")
			for _, feature := range features {
				parts := strings.Split(feature, "=")
				if len(parts) == 2 {
					if val, err := strconv.ParseBool(parts[1]); err == nil {
						mSpec.FeatureGates[parts[0]] = val
					}
				}
			}

			processedArgs[k] = true
		}
	}

	// Note: We don't put unprocessed args into AdditionalArgs here because
	// we can't distinguish between args that were originally in Manager.AdditionalArgs
	// vs args that were originally in Container.Args during round-trip conversion.
	// Instead, unprocessed args will remain in Container.Args.

	// Clean up empty structs
	if len(mSpec.Controller.GroupKindConcurrency) == 0 &&
		mSpec.Controller.CacheSyncTimeout == nil &&
		mSpec.Controller.RecoverPanic == nil {
		mSpec.Controller = nil
	}

	// Return nil if nothing was set
	if mSpec.Controller == nil &&
		mSpec.MaxConcurrentReconciles == 0 &&
		mSpec.CacheNamespace == "" &&
		mSpec.Health.HealthProbeBindAddress == "" &&
		mSpec.LeaderElection == nil &&
		mSpec.Metrics.BindAddress == "" &&
		mSpec.Webhook.Host == "" &&
		mSpec.Webhook.Port == nil &&
		mSpec.Webhook.CertDir == "" &&
		mSpec.SyncPeriod == nil &&
		mSpec.ProfilerAddress == "" &&
		mSpec.Verbosity == 0 &&
		len(mSpec.FeatureGates) == 0 {
		return nil
	}

	return mSpec
}

// Helper to parse duration strings like "30s", "5m", etc.
func parseDurationString(s string) (time.Duration, error) {
	// Handle format like "30s"
	if strings.HasSuffix(s, "s") {
		seconds := strings.TrimSuffix(s, "s")
		if sec, err := strconv.Atoi(seconds); err == nil {
			return time.Duration(sec) * time.Second, nil
		}
	}
	// Try standard duration parsing
	return time.ParseDuration(s)
}
