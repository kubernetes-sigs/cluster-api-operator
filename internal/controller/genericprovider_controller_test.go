/*
Copyright 2021 The Kubernetes Authors.

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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
)

const (
	testMetadata = `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 0
    minor: 4
    contract: v1beta1
`
	testComponents = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cluster.x-k8s.io/provider: infrastructure-docker
    control-plane: controller-manager
  name: capd-controller-manager
  namespace: capd-system
spec:
  replicas: 1
  selector:
    matchLabels:
      cluster.x-k8s.io/provider: infrastructure-docker
      control-plane: controller-manager
  template:
    metadata:
      labels:
        cluster.x-k8s.io/provider: infrastructure-docker
        control-plane: controller-manager
    spec:
      containers:
      - image: gcr.io/google-samples/hello-app:1.0
        name: manager
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 200m
`

	testCurrentVersion = "v0.4.2"
)

func insertDummyConfig(provider genericprovider.GenericProvider) {
	spec := provider.GetSpec()
	spec.FetchConfig = &operatorv1.FetchConfiguration{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test": "dummy-config",
			},
		},
	}
	provider.SetSpec(spec)
}

func dummyConfigMap(ns, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"test": "dummy-config",
			},
		},
		Data: map[string]string{
			"metadata":   testMetadata,
			"components": testComponents,
		},
	}
}

func TestReconcilerPreflightConditions(t *testing.T) {
	testCases := []struct {
		name      string
		namespace string
		providers []genericprovider.GenericProvider
	}{
		{
			name:      "preflight conditions for CoreProvider",
			namespace: "test-core-provider",
			providers: []genericprovider.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-api",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: testCurrentVersion,
						},
					},
				},
			},
		},
		{
			name:      "preflight conditions for ControlPlaneProvider",
			namespace: "test-cp-provider",
			providers: []genericprovider.GenericProvider{
				&operatorv1.CoreProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-api",
					},
					Spec: operatorv1.CoreProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: testCurrentVersion,
						},
					},
					Status: operatorv1.CoreProviderStatus{
						ProviderStatus: operatorv1.ProviderStatus{
							Conditions: []clusterv1.Condition{
								{
									Type:   clusterv1.ReadyCondition,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
				&operatorv1.ControlPlaneProvider{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubeadm",
					},
					Spec: operatorv1.ControlPlaneProviderSpec{
						ProviderSpec: operatorv1.ProviderSpec{
							Version: testCurrentVersion,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			t.Log("Ensure namespace exists", tc.namespace)
			g.Expect(env.EnsureNamespaceExists(ctx, tc.namespace)).To(Succeed())

			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(tc.namespace, testCurrentVersion))).To(Succeed())

			for _, p := range tc.providers {
				insertDummyConfig(p)
				p.SetNamespace(tc.namespace)
				t.Log("creating test provider", p.GetName())
				g.Expect(env.CreateAndWait(ctx, p)).To(Succeed())
			}

			g.Eventually(func() bool {
				for _, p := range tc.providers {
					if err := env.Get(ctx, client.ObjectKeyFromObject(p), p); err != nil {
						return false
					}

					for _, cond := range p.GetStatus().Conditions {
						if cond.Type == operatorv1.PreflightCheckCondition {
							t.Log(t.Name(), p.GetName(), cond)
							if cond.Status == corev1.ConditionTrue {
								return true
							}
						}
					}
				}

				return false
			}, timeout).Should(BeEquivalentTo(true))

			objs := []client.Object{}
			for _, p := range tc.providers {
				objs = append(objs, p)
			}

			objs = append(objs, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testCurrentVersion,
					Namespace: tc.namespace,
				},
			})

			g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
		})
	}
}

func TestAirGappedUpgradeDowngradeProvider(t *testing.T) {
	currentVersion := "v999.9.2"
	futureMetadata := `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
  - major: 999
    minor: 9
    contract: v1beta1
`

	dummyFutureConfigMap := func(ns, name string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"test": "dummy-config",
				},
			},
			Data: map[string]string{
				"metadata":   futureMetadata,
				"components": testComponents,
			},
		}
	}

	testCases := []struct {
		name       string
		newVersion string
	}{
		{
			name:       "same provider version",
			newVersion: "v999.9.2",
		},
		{
			name:       "upgrade provider version",
			newVersion: "v999.9.3",
		},
		{
			name:       "downgrade provider version",
			newVersion: "v999.9.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-api",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: currentVersion,
					},
				},
			}

			namespace := "test-upgrades-downgrades"

			t.Log("Ensure namespace exists", namespace)
			g.Expect(env.EnsureNamespaceExists(ctx, namespace)).To(Succeed())

			g.Expect(env.CreateAndWait(ctx, dummyFutureConfigMap(namespace, currentVersion))).To(Succeed())

			insertDummyConfig(provider)
			provider.SetNamespace(namespace)
			t.Log("creating test provider", provider.GetName())
			g.Expect(env.CreateAndWait(ctx, provider)).To(Succeed())

			g.Eventually(func() bool {
				if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
					return false
				}

				if provider.GetStatus().InstalledVersion == nil || *provider.GetStatus().InstalledVersion != currentVersion {
					return false
				}

				for _, cond := range provider.GetStatus().Conditions {
					if cond.Type == operatorv1.PreflightCheckCondition {
						t.Log(t.Name(), provider.GetName(), cond)
						if cond.Status == corev1.ConditionTrue {
							return true
						}
					}
				}

				return false
			}, timeout).Should(BeEquivalentTo(true))

			// creating another configmap with another version
			if tc.newVersion != currentVersion {
				g.Expect(env.CreateAndWait(ctx, dummyFutureConfigMap(namespace, tc.newVersion))).To(Succeed())
			}

			// Change provider version
			providerSpec := provider.GetSpec()
			providerSpec.Version = tc.newVersion
			providerSpec.Deployment = &operatorv1.DeploymentSpec{
				Replicas: ptr.To(2),
			}
			provider.SetSpec(providerSpec)

			// Set label (needed to start a reconciliation of the provider)
			labels := provider.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}

			labels["provider-version"] = tc.newVersion
			provider.SetLabels(labels)

			g.Expect(env.Client.Update(ctx, provider)).To(Succeed())

			g.Eventually(func() bool {
				if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
					return false
				}

				if provider.GetStatus().InstalledVersion == nil || *provider.GetStatus().InstalledVersion != tc.newVersion {
					return false
				}

				if provider.GetLabels()["provider-version"] != tc.newVersion {
					return false
				}

				allFound := false
				for _, cond := range provider.GetStatus().Conditions {
					if cond.Type == operatorv1.PreflightCheckCondition {
						t.Log(t.Name(), provider.GetName(), cond)
						if cond.Status == corev1.ConditionTrue {
							allFound = true
							break
						}
					}
				}

				if !allFound {
					return false
				}

				allFound = tc.newVersion == currentVersion
				for _, cond := range provider.GetStatus().Conditions {
					if cond.Type == operatorv1.ProviderUpgradedCondition {
						t.Log(t.Name(), provider.GetName(), cond)
						if cond.Status == corev1.ConditionTrue {
							allFound = tc.newVersion != currentVersion
							break
						}
					}
				}

				if !allFound {
					return false
				}

				// Ensure customization occurred
				dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
					Namespace: provider.Namespace,
					Name:      "capd-controller-manager",
				}}
				if err := env.Get(ctx, client.ObjectKeyFromObject(dep), dep); err != nil {
					return false
				}

				return dep.Spec.Replicas != nil && *dep.Spec.Replicas == 2
			}, timeout).Should(BeEquivalentTo(true))

			g.Consistently(func() bool {
				allSet := tc.newVersion == currentVersion
				for _, cond := range provider.GetStatus().Conditions {
					if cond.Type == operatorv1.ProviderUpgradedCondition {
						t.Log(t.Name(), provider.GetName(), cond)
						if cond.Status == corev1.ConditionTrue {
							allSet = tc.newVersion != currentVersion
							break
						}
					}
				}

				return allSet
			}, 2*time.Second).Should(BeTrue())

			// Clean up
			objs := []client.Object{provider}
			objs = append(objs, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      currentVersion,
					Namespace: namespace,
				},
			})

			objs = append(objs, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.newVersion,
					Namespace: namespace,
				},
			})

			g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
		})
	}
}

func TestProviderSpecChanges(t *testing.T) {
	testCases := []struct {
		name        string
		spec        operatorv1.ProviderSpec
		updatedSpec operatorv1.ProviderSpec
		expectError bool
	}{
		{
			name: "same spec, hash annotation doesn't change",
			spec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
			updatedSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
		},
		{
			name: "add more replicas, hash annotation is updated",
			spec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
			updatedSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				Deployment: &operatorv1.DeploymentSpec{
					Replicas: ptr.To(2),
				},
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
		},
		{
			name:        "upgrade to a non-existent version, hash annotation is empty",
			expectError: true,
			spec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
			updatedSpec: operatorv1.ProviderSpec{
				Version: "10000.0.0-NONEXISTENT",
				Deployment: &operatorv1.DeploymentSpec{
					Replicas: ptr.To(2),
				},
				FetchConfig: &operatorv1.FetchConfiguration{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "dummy-config",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			specHash, err := calculateHash(tc.spec)
			g.Expect(err).ToNot(HaveOccurred())

			updatedSpecHash, err := calculateHash(tc.updatedSpec)
			g.Expect(err).ToNot(HaveOccurred())

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-api",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: tc.spec,
				},
			}

			namespace := "test-provider-spec-changes"

			ns, err := env.CreateNamespace(ctx, namespace)
			t.Log("Ensure namespace exists", ns.Name)
			g.Expect(err).ToNot(HaveOccurred())

			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(ns.Name, testCurrentVersion))).To(Succeed())

			provider.SetNamespace(ns.Name)
			t.Log("creating test provider", provider.GetName())
			g.Expect(env.CreateAndWait(ctx, provider.DeepCopy())).To(Succeed())

			g.Eventually(generateExpectedResultChecker(provider, specHash, corev1.ConditionTrue), timeout).Should(BeEquivalentTo(true))

			g.Eventually(func() error {
				if err := env.Client.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
					return err
				}

				// Change provider spec
				provider.SetSpec(tc.updatedSpec)

				// Set a label to ensure that provider was changed
				labels := provider.GetLabels()
				if labels == nil {
					labels = map[string]string{}
				}
				labels["my-label"] = "some-value"
				provider.SetLabels(labels)
				provider.SetManagedFields(nil)

				return env.Client.Update(ctx, provider)
			}).Should(Succeed())

			if !tc.expectError {
				g.Eventually(generateExpectedResultChecker(provider, updatedSpecHash, corev1.ConditionTrue), timeout).Should(BeEquivalentTo(true))
			} else {
				g.Eventually(generateExpectedResultChecker(provider, "", corev1.ConditionFalse), timeout).Should(BeEquivalentTo(true))
			}

			// Clean up
			g.Expect(env.Cleanup(ctx, provider, ns)).To(Succeed())
		})
	}
}

func generateExpectedResultChecker(provider genericprovider.GenericProvider, specHash string, condStatus corev1.ConditionStatus) func() bool {
	return func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		// In case of error we don't want the spec annotation to be updated
		if provider.GetAnnotations()[appliedSpecHashAnnotation] != specHash {
			return false
		}

		for _, cond := range provider.GetStatus().Conditions {
			if cond.Type == operatorv1.ProviderInstalledCondition {
				if cond.Status == condStatus {
					return true
				}
			}
		}

		return false
	}
}

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme))

	return scheme
}
