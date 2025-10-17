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

package v1alpha2_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	randfill "sigs.k8s.io/randfill"

	operatorv1alpha2 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

// The fuzzer functions need to be updated to work with the latest randfill API.
func TestFuzzyConversion(t *testing.T) {
	t.Run("for CoreProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.CoreProvider{},
		Spoke:       &operatorv1alpha2.CoreProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for BootstrapProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.BootstrapProvider{},
		Spoke:       &operatorv1alpha2.BootstrapProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for ControlPlaneProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.ControlPlaneProvider{},
		Spoke:       &operatorv1alpha2.ControlPlaneProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for InfrastructureProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.InfrastructureProvider{},
		Spoke:       &operatorv1alpha2.InfrastructureProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for AddonProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.AddonProvider{},
		Spoke:       &operatorv1alpha2.AddonProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for IPAMProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.IPAMProvider{},
		Spoke:       &operatorv1alpha2.IPAMProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))

	t.Run("for RuntimeExtensionProvider", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &operatorv1.RuntimeExtensionProvider{},
		Spoke:       &operatorv1alpha2.RuntimeExtensionProvider{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{fuzzFuncs},
	}))
}

func fuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		// Use randfill.Continue as required by randfill
		func(providerSpec *operatorv1alpha2.ProviderSpec, c randfill.Continue) {
			c.FillNoCustom(providerSpec)

			manager := providerSpec.Manager

			// Ensure we have valid test data for Manager fields
			if manager != nil {
				// Set some realistic values for Manager fields
				if manager.MaxConcurrentReconciles == 0 {
					providerSpec.Manager.MaxConcurrentReconciles = c.Intn(10) + 1
				}
				if manager.Verbosity == 0 {
					providerSpec.Manager.Verbosity = c.Intn(5)
				}

				// Ensure feature gates have valid names (no commas or equals signs) and boolean values
				if manager.FeatureGates != nil {
					// Clean up feature gate names to remove problematic characters
					cleanedFeatureGates := make(map[string]bool)
					for fg, val := range manager.FeatureGates {
						// Remove commas and equals signs from feature gate names
						// These characters are used as delimiters in the serialization format
						cleanedName := strings.ReplaceAll(fg, ",", "")
						cleanedName = strings.ReplaceAll(cleanedName, "=", "")
						if cleanedName != "" {
							cleanedFeatureGates[cleanedName] = val
						}
					}
					manager.FeatureGates = cleanedFeatureGates
				}
				if len(manager.FeatureGates) == 0 {
					manager.FeatureGates = map[string]bool{
						"TestFeature": c.Bool(),
					}
				}

				if len(manager.AdditionalArgs) == 0 {
					manager.AdditionalArgs = nil
				}

				if isControllerConfigurationSpecv1alpha2Empty(manager.Controller) {
					manager.Controller = nil
				}

				if manager.Controller != nil && manager.Controller.GroupKindConcurrency != nil {
					if len(manager.Controller.GroupKindConcurrency) == 0 {
						manager.Controller.GroupKindConcurrency = nil
					} else {
						groupKindConcurrency := make(map[string]int)
						for k, v := range manager.Controller.GroupKindConcurrency {
							groupKindConcurrency[strings.ToLower(k)] = v
						}
						manager.Controller.GroupKindConcurrency = groupKindConcurrency
					}
				}

				if manager.LeaderElection != nil {
					manager.LeaderElection.ResourceName = "test-leader"
					manager.LeaderElection.ResourceNamespace = "test-namespace"
				}
			}

			// Ignore Manager spec for additional deployments
			if providerSpec.AdditionalDeployments != nil {
				for k, v := range providerSpec.AdditionalDeployments {
					ad := v // Create a copy to modify

					// Clear Manager field
					if ad.Manager != nil {
						ad.Manager = nil
					}

					// Clear empty containers
					if ad.Deployment != nil && len(ad.Deployment.Containers) == 0 {
						ad.Deployment.Containers = nil
					}

					// Set Deployment to nil if it's empty
					if operatorv1alpha2.IsV1alpha2DeploymentEmpty(ad.Deployment) {
						ad.Deployment = nil
					}

					// Save the modified value back to the map
					providerSpec.AdditionalDeployments[k] = ad
				}
			}

			if providerSpec.Deployment != nil {
				if len(providerSpec.Deployment.Containers) != 0 {
					for i := range providerSpec.Deployment.Containers {
						if providerSpec.Deployment.Containers[i].Name == operatorv1alpha2.DefaultManagerContainerName {
							// Generate a random name
							providerSpec.Deployment.Containers[i].Name = "test-manager-" + c.String(10)
						}
					}
				} else {
					providerSpec.Deployment.Containers = nil
				}

				if operatorv1alpha2.IsV1alpha2DeploymentEmpty(providerSpec.Deployment) {
					providerSpec.Deployment = nil
				}
			}
		},
		func(providerSpec *operatorv1.ProviderSpec, c randfill.Continue) {
			c.FillNoCustom(providerSpec)

			// Ensure container args are valid
			if providerSpec.Deployment != nil && len(providerSpec.Deployment.Containers) > 0 {
				for i := range providerSpec.Deployment.Containers {
					if providerSpec.Deployment.Containers[i].Args == nil {
						providerSpec.Deployment.Containers[i].Args = make(map[string]string)
					}
					// Add some realistic args
					if c.Bool() {
						providerSpec.Deployment.Containers[i].Args["--v"] = "2"
					}
					if c.Bool() {
						providerSpec.Deployment.Containers[i].Args["--max-concurrent-reconciles"] = "5"
					}
				}
			}

			if providerSpec.Deployment != nil && len(providerSpec.Deployment.Containers) != 0 {
				for i := range providerSpec.Deployment.Containers {
					if providerSpec.Deployment.Containers[i].Name == operatorv1alpha2.DefaultManagerContainerName {
						providerSpec.Deployment.Containers[i].Name = "test-manager-" + c.String(10)
					}
				}
			}

			if providerSpec.Deployment != nil && operatorv1.IsV1alpha3DeploymentEmpty(providerSpec.Deployment) {
				providerSpec.Deployment = nil
			}
		},
	}
}

func isControllerConfigurationSpecv1alpha2Empty(c *operatorv1alpha2.ControllerConfigurationSpec) bool {
	if c == nil {
		return true
	}

	return len(c.GroupKindConcurrency) == 0 && c.CacheSyncTimeout == nil && c.RecoverPanic == nil
}

func TestManagerSpecToArgsConversion(t *testing.T) {
	g := NewWithT(t)

	t.Run("should convert ManagerSpec to container args", func(t *testing.T) {
		leaderElect := true
		port := 9443
		replicas := 1

		v2Provider := &operatorv1alpha2.CoreProvider{
			Spec: operatorv1alpha2.CoreProviderSpec{
				ProviderSpec: operatorv1alpha2.ProviderSpec{
					Version: "v1.0.0",
					Manager: &operatorv1alpha2.ManagerSpec{
						ControllerManagerConfiguration: operatorv1alpha2.ControllerManagerConfiguration{
							SyncPeriod: &metav1.Duration{Duration: 30 * time.Second},
							LeaderElection: &configv1alpha1.LeaderElectionConfiguration{
								LeaderElect:       &leaderElect,
								ResourceNamespace: "test-namespace",
								ResourceName:      "test-leader",
							},
							CacheNamespace: "test-cache-ns",
							Health: operatorv1alpha2.ControllerHealth{
								HealthProbeBindAddress: ":8081",
							},
							Metrics: operatorv1alpha2.ControllerMetrics{
								BindAddress: ":8080",
							},
							Webhook: operatorv1alpha2.ControllerWebhook{
								Host:    "webhook.example.com",
								Port:    &port,
								CertDir: "/tmp/certs",
							},
						},
						ProfilerAddress:         "localhost:6060",
						MaxConcurrentReconciles: 10,
						Verbosity:               2,
						FeatureGates: map[string]bool{
							"FeatureA": true,
							"FeatureB": false,
						},
						AdditionalArgs: map[string]string{
							"--custom-arg": "custom-value",
						},
					},
					Deployment: &operatorv1alpha2.DeploymentSpec{
						Replicas: &replicas,
						Containers: []operatorv1alpha2.ContainerSpec{
							{
								Name: "some-container",
								Args: map[string]string{
									"--some-arg": "some-value",
								},
							},
						},
					},
				},
			},
		}

		// Convert to v3
		v3Provider := &operatorv1.CoreProvider{}
		err := v2Provider.ConvertTo(v3Provider)
		g.Expect(err).NotTo(HaveOccurred())

		// Check that Manager fields were converted to container args
		g.Expect(v3Provider.Spec.Deployment).NotTo(BeNil())
		g.Expect(v3Provider.Spec.Deployment.Containers).To(HaveLen(2))

		args := v3Provider.Spec.Deployment.Containers[1].Args
		g.Expect(args).NotTo(BeNil())

		// Check converted args
		g.Expect(args["--sync-period"]).To(Equal("30s"))
		g.Expect(args["--leader-elect"]).To(Equal("true"))
		g.Expect(args["--leader-election-id"]).To(Equal("test-namespace/test-leader"))
		g.Expect(args["--namespace"]).To(Equal("test-cache-ns"))
		g.Expect(args["--health-addr"]).To(Equal(":8081"))
		g.Expect(args["--metrics-bind-addr"]).To(Equal(":8080"))
		g.Expect(args["--webhook-host"]).To(Equal("webhook.example.com"))
		g.Expect(args["--webhook-port"]).To(Equal("9443"))
		g.Expect(args["--webhook-cert-dir"]).To(Equal("/tmp/certs"))
		g.Expect(args["--profiler-address"]).To(Equal("localhost:6060"))
		g.Expect(args["--max-concurrent-reconciles"]).To(Equal("10"))
		g.Expect(args["--v"]).To(Equal("2"))
		g.Expect(args["--feature-gates"]).To(ContainSubstring("FeatureA=true"))
		g.Expect(args["--feature-gates"]).To(ContainSubstring("FeatureB=false"))
		g.Expect(args["--custom-arg"]).To(Equal("custom-value"))

		args = v3Provider.Spec.Deployment.Containers[0].Args
		g.Expect(args).NotTo(BeNil())
		g.Expect(args["--some-arg"]).To(Equal("some-value"))

		// Convert back to v2
		v2ProviderConverted := &operatorv1alpha2.CoreProvider{}
		err = v2ProviderConverted.ConvertFrom(v3Provider)
		g.Expect(err).NotTo(HaveOccurred())

		// Check that Manager was restored
		g.Expect(v2ProviderConverted.Spec.Manager).NotTo(BeNil())
		g.Expect(v2ProviderConverted.Spec.Manager.MaxConcurrentReconciles).To(Equal(10))
		g.Expect(v2ProviderConverted.Spec.Manager.Verbosity).To(Equal(2))
		g.Expect(v2ProviderConverted.Spec.Manager.ProfilerAddress).To(Equal("localhost:6060"))
		g.Expect(v2ProviderConverted.Spec.Manager.FeatureGates).To(HaveLen(2))
		g.Expect(v2ProviderConverted.Spec.Manager.FeatureGates["FeatureA"]).To(BeTrue())
		g.Expect(v2ProviderConverted.Spec.Manager.FeatureGates["FeatureB"]).To(BeFalse())
	})

	t.Run("should handle GroupKindConcurrency conversion", func(t *testing.T) {
		v2Provider := &operatorv1alpha2.CoreProvider{
			Spec: operatorv1alpha2.CoreProviderSpec{
				ProviderSpec: operatorv1alpha2.ProviderSpec{
					Manager: &operatorv1alpha2.ManagerSpec{
						ControllerManagerConfiguration: operatorv1alpha2.ControllerManagerConfiguration{
							Controller: &operatorv1alpha2.ControllerConfigurationSpec{
								GroupKindConcurrency: map[string]int{
									"Cluster": 10,
									"Machine": 5,
								},
							},
						},
					},
					Deployment: &operatorv1alpha2.DeploymentSpec{
						Containers: []operatorv1alpha2.ContainerSpec{
							{
								Name: "manager",
							},
						},
					},
				},
			},
		}

		// Convert to v3
		v3Provider := &operatorv1.CoreProvider{}
		err := v2Provider.ConvertTo(v3Provider)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(v3Provider.Spec.Deployment.Containers[1].Name).To(BeEmpty())

		args := v3Provider.Spec.Deployment.Containers[1].Args
		g.Expect(args).NotTo(BeNil())
		g.Expect(args["--cluster-concurrency"]).To(Equal("10"))
		g.Expect(args["--machine-concurrency"]).To(Equal("5"))

		// Convert back to v2
		v2ProviderConverted := &operatorv1alpha2.CoreProvider{}
		err = v2ProviderConverted.ConvertFrom(v3Provider)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(v2ProviderConverted.Spec.Manager).NotTo(BeNil())
		g.Expect(v2ProviderConverted.Spec.Manager.Controller).NotTo(BeNil())
		g.Expect(v2ProviderConverted.Spec.Manager.Controller.GroupKindConcurrency).To(HaveLen(2))
		g.Expect(v2ProviderConverted.Spec.Manager.Controller.GroupKindConcurrency["cluster"]).To(Equal(10))
		g.Expect(v2ProviderConverted.Spec.Manager.Controller.GroupKindConcurrency["machine"]).To(Equal(5))
	})
}

// Test helpers to ensure roundtrip conversions work correctly.
func TestConversionRoundTrip(t *testing.T) {
	scheme := runtime.NewScheme()
	g := NewWithT(t)

	g.Expect(operatorv1alpha2.AddToScheme(scheme)).To(Succeed())
	g.Expect(operatorv1.AddToScheme(scheme)).To(Succeed())

	t.Run("CoreProvider round trip", func(t *testing.T) {
		original := &operatorv1alpha2.CoreProvider{
			Spec: operatorv1alpha2.CoreProviderSpec{
				ProviderSpec: operatorv1alpha2.ProviderSpec{
					Version: "v1.0.0",
				},
			},
		}

		hub := &operatorv1.CoreProvider{}
		g.Expect(original.ConvertTo(hub)).To(Succeed())

		spoke := &operatorv1alpha2.CoreProvider{}
		g.Expect(spoke.ConvertFrom(hub)).To(Succeed())

		// Basic check that version survived the round trip
		g.Expect(spoke.Spec.Version).To(Equal(original.Spec.Version))
	})
}
