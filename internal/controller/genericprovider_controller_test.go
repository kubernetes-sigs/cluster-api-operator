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
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
)

const (
	testMetadata = `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 1
    minor: 11
    contract: v1beta2
`
	testDeploymentName = "capd-controller-manager"
	testComponents     = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    cluster.x-k8s.io/provider: infrastructure-docker
    control-plane: controller-manager
    value-from-config: ${CONFIGURED_VALUE:=default-value}
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

	testCurrentVersion = "v1.11.0"
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

func dummyConfigMap(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCurrentVersion,
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

func createDummyProviderWithConfigSecret(objs []client.Object, provider genericprovider.GenericProvider, configSecret *corev1.Secret) ([]client.Object, error) {
	cm := dummyConfigMap(provider.GetNamespace())

	if err := env.CreateAndWait(ctx, cm); err != nil {
		return objs, err
	}

	objs = append(objs, cm)

	provider.SetSpec(operatorv1.ProviderSpec{
		Version: testCurrentVersion,
		ConfigSecret: &operatorv1.SecretReference{
			Name:      configSecret.GetName(),
			Namespace: configSecret.GetNamespace(),
		},
		ManifestPatches: []string{},
	})

	insertDummyConfig(provider)

	err := env.CreateAndWait(ctx, provider)
	if err != nil {
		return objs, err
	}

	objs = append(objs, provider)

	return objs, nil
}

func testDeploymentLabelValueGetter(deploymentNS, deploymentName string) func() string {
	return func() string {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: deploymentNS,
			},
		}

		if err := env.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
			return ""
		}

		return deployment.Labels["value-from-config"]
	}
}

func TestConfigSecretChangesAreAppliedToTheDeployment(t *testing.T) {
	g := NewWithT(t)
	objs := []client.Object{}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
	}()

	ns, err := env.CreateNamespace(ctx, "config-secret-namespace")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	t.Log("Ensure namespace exists", ns.Name)

	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "config-secret-",
			Namespace:    ns.Name,
		},
		StringData: map[string]string{
			"CONFIGURED_VALUE": "initial-value",
		},
	}
	g.Expect(env.CreateAndWait(ctx, configSecret)).To(Succeed())
	objs = append(objs, configSecret)

	t.Log("Created config secret")

	objs, err = createDummyProviderWithConfigSecret(
		objs,
		&operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-api",
				Namespace: ns.Name,
			},
		},
		configSecret,
	)
	g.Expect(err).To(Succeed())

	t.Log("Created provider")

	g.Eventually(
		testDeploymentLabelValueGetter(ns.Name, testDeploymentName),
		timeout,
	).Should(BeEquivalentTo("initial-value"))

	t.Log("Provider deployment deployed")

	configSecret.Data["CONFIGURED_VALUE"] = []byte("updated-value")

	g.Expect(env.Update(ctx, configSecret)).NotTo(HaveOccurred())

	t.Log("Config secret updated")

	g.Eventually(
		testDeploymentLabelValueGetter(ns.Name, testDeploymentName),
		timeout,
	).Should(BeEquivalentTo("updated-value"))
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
							Conditions: []metav1.Condition{
								{
									Type:   clusterv1.ReadyCondition,
									Status: metav1.ConditionTrue,
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

			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(tc.namespace))).To(Succeed())

			for _, p := range tc.providers {
				insertDummyConfig(p)
				p.SetNamespace(tc.namespace)
				t.Log("creating test provider", p.GetName())
				g.Expect(env.CreateAndWait(ctx, p)).To(Succeed())
			}

			defer func() {
				objs := []client.Object{
					dummyConfigMap(tc.namespace),
				}
				for _, p := range tc.providers {
					objs = append(objs, p)
				}

				g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
			}()

			g.Eventually(func() bool {
				for _, p := range tc.providers {
					if err := env.Get(ctx, client.ObjectKeyFromObject(p), p); err != nil {
						return false
					}

					if conditions.IsTrue(p, operatorv1.PreflightCheckCondition) {
						return true
					}
				}

				return false
			}, timeout).Should(BeEquivalentTo(true))
		})
	}
}

func TestAirGappedUpgradeDowngradeProvider(t *testing.T) {
	currentVersion := "v999.9.2"
	futureMetadata := `
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 999
    minor: 9
    contract: v1beta2
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

			defer func() {
				// Clean up
				g.Expect(env.CleanupAndWait(ctx, []client.Object{provider, dummyFutureConfigMap(namespace, currentVersion), dummyFutureConfigMap(namespace, tc.newVersion)}...)).To(Succeed())
			}()

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
						if cond.Status == metav1.ConditionTrue {
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
						if cond.Status == metav1.ConditionTrue {
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
						if cond.Status == metav1.ConditionTrue {
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
					Name:      testDeploymentName,
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
						if cond.Status == metav1.ConditionTrue {
							allSet = tc.newVersion != currentVersion
							break
						}
					}
				}

				return allSet
			}, 2*time.Second).Should(BeTrue())
		})
	}
}

func TestProviderShouldNotBeInstalledWhenCoreProviderNotReady(t *testing.T) {
	g := NewGomegaWithT(t)
	coreProvider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-api",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: "v0.0.1-incorrect",
			},
		},
	}
	controlPlaneProvider := &operatorv1.ControlPlaneProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeadm",
		},
		Spec: operatorv1.ControlPlaneProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}

	ns, err := env.CreateNamespace(ctx, "core-provider-not-ready")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ns.Name).NotTo(BeEmpty())

	coreProvider.Namespace = ns.Name
	g.Expect(env.CreateAndWait(ctx, coreProvider)).To(Succeed())

	controlPlaneProvider.Namespace = ns.Name
	g.Expect(env.CreateAndWait(ctx, controlPlaneProvider)).To(Succeed())

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, []client.Object{coreProvider, controlPlaneProvider}...)).To(Succeed())
	}()

	g.Eventually(func() bool {
		if err = env.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider); err != nil {
			return false
		}

		return conditions.Has(coreProvider, operatorv1.ProviderInstalledCondition) && conditions.IsFalse(coreProvider, operatorv1.ProviderInstalledCondition)
	}, timeout).Should(BeEquivalentTo(true))

	g.Consistently(func() bool {
		if err = env.Get(ctx, client.ObjectKeyFromObject(controlPlaneProvider), controlPlaneProvider); err != nil {
			return false
		}

		return !conditions.IsTrue(controlPlaneProvider, operatorv1.PreflightCheckCondition) &&
			!conditions.IsTrue(controlPlaneProvider, operatorv1.ProviderInstalledCondition) &&
			!conditions.IsTrue(controlPlaneProvider, clusterv1.ReadyCondition)
	}, timeout/3).Should(BeEquivalentTo(true))
}

func TestReconcilerPreflightConditionsFromCoreProviderEvents(t *testing.T) {
	namespace := "test-core-provider-events"
	coreProvider := &operatorv1.CoreProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-api",
		},
		Spec: operatorv1.CoreProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}
	infrastructureProvider := &operatorv1.InfrastructureProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vsphere",
		},
		Spec: operatorv1.InfrastructureProviderSpec{
			ProviderSpec: operatorv1.ProviderSpec{
				Version: testCurrentVersion,
			},
		},
	}

	g := NewWithT(t)
	t.Log("Ensure namespace exists", namespace)
	g.Expect(env.EnsureNamespaceExists(ctx, namespace)).To(Succeed())

	g.Expect(env.CreateAndWait(ctx, dummyConfigMap(namespace))).To(Succeed())

	for _, p := range []genericprovider.GenericProvider{coreProvider, infrastructureProvider} {
		insertDummyConfig(p)
		p.SetNamespace(namespace)
		t.Log("creating test provider", p.GetName())
		g.Expect(env.CreateAndWait(ctx, p)).To(Succeed())
	}

	defer func() {
		g.Expect(env.CleanupAndWait(ctx, []client.Object{coreProvider, infrastructureProvider, dummyConfigMap(namespace)}...)).To(Succeed())
	}()

	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(infrastructureProvider), infrastructureProvider); err != nil {
			return false
		}

		if conditions.Has(infrastructureProvider, operatorv1.PreflightCheckCondition) && conditions.IsFalse(infrastructureProvider, operatorv1.PreflightCheckCondition) {
			return true
		}

		return false
	}, timeout).Should(BeEquivalentTo(true))

	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider); err != nil {
			return false
		}

		if conditions.IsTrue(coreProvider, operatorv1.PreflightCheckCondition) && conditions.IsTrue(coreProvider, operatorv1.ProviderInstalledCondition) {
			return true
		}

		return false
	}, timeout).Should(BeEquivalentTo(true))

	patchHelper, err := patch.NewHelper(coreProvider, env)
	g.Expect(err).ToNot(HaveOccurred())

	conditions.Set(coreProvider, metav1.Condition{
		Type:    clusterv1.ReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  "Ready",
		Message: "Provider is ready",
	})
	g.Expect(patchHelper.Patch(ctx, coreProvider)).To(Succeed())

	g.Eventually(func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(infrastructureProvider), infrastructureProvider); err != nil {
			return false
		}

		if conditions.IsTrue(infrastructureProvider, operatorv1.PreflightCheckCondition) {
			return true
		}

		return false
	}, timeout).Should(BeEquivalentTo(true))
}

func TestProviderConfigSecretChanges(t *testing.T) {
	testCases := []struct {
		name           string
		cmData         map[string][]byte
		updatedCMData  map[string][]byte
		expectSameHash bool
	}{
		{
			name: "With the same config secret data, the hash annotation doesn't change",
			cmData: map[string][]byte{
				"some-key":    []byte("some data"),
				"another-key": []byte("another data"),
			},
			updatedCMData: map[string][]byte{
				"another-key": []byte("another data"),
				"some-key":    []byte("some data"),
			},
			expectSameHash: true,
		},
		{
			name: "With different config secret data, the hash annotation changes",
			cmData: map[string][]byte{
				"some-key":    []byte("some data"),
				"another-key": []byte("another data"),
			},
			updatedCMData: map[string][]byte{
				"another-key": []byte("another data"),
				"some-key":    []byte("some updated data"),
			},
			expectSameHash: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			objs := []client.Object{}

			defer func() {
				g.Expect(env.CleanupAndWait(ctx, objs...)).To(Succeed())
			}()

			cmSecretName := "test-config"

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-api",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						Version: testCurrentVersion,
						FetchConfig: &operatorv1.FetchConfiguration{
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"test": "dummy-config",
								},
							},
						},
						ConfigSecret: &operatorv1.SecretReference{
							Name: cmSecretName,
						},
					},
				},
			}

			providerNamespace, err := env.CreateNamespace(ctx, "test-provider")
			t.Log("Ensure namespace exists", providerNamespace.Name)
			g.Expect(err).ToNot(HaveOccurred())

			configNamespace, err := env.CreateNamespace(ctx, "test-provider-config-changes")
			t.Log("Ensure namespace exists", configNamespace.Name)
			g.Expect(err).ToNot(HaveOccurred())

			cm := dummyConfigMap(providerNamespace.Name)
			g.Expect(env.CreateAndWait(ctx, cm)).To(Succeed())
			objs = append(objs, cm)

			provider.Namespace = providerNamespace.Name
			provider.Spec.ConfigSecret.Namespace = configNamespace.Name

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: configNamespace.Name,
					Name:      cmSecretName,
				},
				Data: tc.cmData,
			}

			g.Expect(env.CreateAndWait(ctx, secret.DeepCopy())).To(Succeed())
			objs = append(objs, secret)

			t.Log("creating test provider", provider.GetName())
			g.Expect(env.CreateAndWait(ctx, provider.DeepCopy())).To(Succeed())
			objs = append(objs, provider)

			g.Eventually(generateExpectedResultChecker(provider, metav1.ConditionTrue, func(s string) bool { return s != "" }), timeout).Should(BeEquivalentTo(true))

			initialHash := provider.GetAnnotations()[appliedSpecHashAnnotation]

			g.Eventually(func() error {
				if err := env.Client.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
					return err
				}

				// Change provider config data
				secret.Data = tc.updatedCMData

				return env.Client.Update(ctx, secret)
			}).Should(Succeed())

			if tc.expectSameHash {
				g.Consistently(func(g Gomega) {
					err := env.Client.Get(ctx, client.ObjectKeyFromObject(provider), provider)
					g.Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())

					g.Expect(provider.GetAnnotations()[appliedSpecHashAnnotation]).To(Equal(initialHash))
				}, 15*time.Second).Should(Succeed())
			} else {
				g.Eventually(func(g Gomega) {
					err := env.Client.Get(ctx, client.ObjectKeyFromObject(provider), provider)
					g.Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())

					g.Expect(provider.GetAnnotations()[appliedSpecHashAnnotation]).ToNot(Equal(initialHash))
				}, 15*time.Second).ShouldNot(Succeed())
			}
		})
	}
}

func TestProviderSpecChanges(t *testing.T) {
	testCases := []struct {
		name             string
		spec             operatorv1.ProviderSpec
		updatedSpec      operatorv1.ProviderSpec
		expectHashChange bool
		expectError      bool
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
			expectHashChange: true,
		},
		{
			name:        "upgrade to a non-existent version, hash annotation unchanged",
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

			provider := &operatorv1.CoreProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-api",
				},
				Spec: operatorv1.CoreProviderSpec{
					ProviderSpec: tc.spec,
				},
			}

			updatedProvider := provider.DeepCopy()
			updatedProvider.SetSpec(tc.updatedSpec)

			namespace := "test-provider-spec-changes"

			ns, err := env.CreateNamespace(ctx, namespace)
			t.Log("Ensure namespace exists", ns.Name)
			g.Expect(err).ToNot(HaveOccurred())

			g.Expect(env.CreateAndWait(ctx, dummyConfigMap(ns.Name))).To(Succeed())

			provider.SetNamespace(ns.Name)
			t.Log("creating test provider", provider.GetName())
			g.Expect(env.CreateAndWait(ctx, provider.DeepCopy())).To(Succeed())

			defer func() {
				// Clean up
				g.Expect(env.Cleanup(ctx, provider, dummyConfigMap(namespace))).To(Succeed())
			}()

			g.Eventually(generateExpectedResultChecker(provider, metav1.ConditionTrue, func(s string) bool {
				return s != ""
			}), timeout).Should(BeEquivalentTo(true))

			currentHash := provider.GetAnnotations()[appliedSpecHashAnnotation]

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
				g.Eventually(generateExpectedResultChecker(provider, metav1.ConditionTrue, func(s string) bool {
					if tc.expectHashChange {
						return s != currentHash
					}

					return s == currentHash
				}), timeout).Should(BeEquivalentTo(true))
			} else {
				g.Eventually(generateExpectedResultChecker(provider, metav1.ConditionFalse, func(s string) bool { return s == currentHash }), timeout).Should(BeEquivalentTo(true))
			}
		})
	}
}

func generateExpectedResultChecker(provider genericprovider.GenericProvider, condStatus metav1.ConditionStatus, hashCheck func(string) bool) func() bool {
	return func() bool {
		if err := env.Get(ctx, client.ObjectKeyFromObject(provider), provider); err != nil {
			return false
		}

		// In case of error we don't want the spec annotation to be updated
		if !hashCheck(provider.GetAnnotations()[appliedSpecHashAnnotation]) {
			return false
		}

		condition := conditions.Get(provider, operatorv1.ProviderInstalledCondition)

		return condition != nil && condition.Status == condStatus
	}
}

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme))

	return scheme
}
