//go:build e2e

/*
Copyright 2022 The Kubernetes Authors.

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

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api/test/framework"

	. "sigs.k8s.io/cluster-api-operator/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mediaType    = "application/vnd.test.file"
	artifactType = "application/vnd.acme.config"
)

var _ = Describe("Create, upgrade, downgrade and delete providers with minimal specified configuration", func() {
	It("should successfully create a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()

		additionalManifestsCMName := "additional-manifests"
		additionalManifests := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      additionalManifestsCMName,
				Namespace: operatorNamespace,
			},
			Data: map[string]string{
				"manifests": `
---				
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config-map
  namespace: capi-operator-system
data: 
  test: test
`,
			},
		}

		Expect(bootstrapCluster.Create(ctx, additionalManifests)).To(Succeed())

		coreProvider := &operatorv1.CoreProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.CoreProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					AdditionalManifestsRef: &operatorv1.ConfigmapReference{
						Name: additionalManifests.Name,
					},
					ManifestPatches: []string{`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    test-label: test-value`},
				},
			},
		}

		Expect(bootstrapCluster.Create(ctx, coreProvider)).To(Succeed())
		Expect(bootstrapCluster.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())
		Expect(coreProvider.Spec.AdditionalManifestsRef).ToNot(BeNil())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Checking for deployment to have additional labels")
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}}
		WaitFor(ctx, For(deployment).In(bootstrapCluster).ToSatisfy(func() bool {
			if v, ok := deployment.Labels["test-label"]; ok {
				return v == "test-value"
			}

			return false
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, &coreProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Checking if additional manifests are applied")
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-config-map",
				Namespace: operatorNamespace,
			},
		}
		WaitFor(ctx, For(cm).In(bootstrapCluster).ToSatisfy(func() bool {
			value, ok := cm.Data["test"]
			return ok && value == "test"
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete a BootstrapProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		bootstrapProvider := &operatorv1.BootstrapProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapProviderDeploymentName,
			Namespace: operatorNamespace,
		}}

		Expect(bootstrapCluster.Create(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for bootstrap provider to be ready")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&bootstrapProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(bootstrapProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(bootstrapProvider.Status.InstalledVersion, &bootstrapProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
		Expect(bootstrapCluster.Delete(ctx, bootstrapProvider)).To(Succeed())

		By("Waiting for the bootstrap provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete a ControlPlaneProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		cpProvider := &operatorv1.ControlPlaneProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      cpProviderDeploymentName,
			Namespace: operatorNamespace,
		}}

		Expect(bootstrapCluster.Create(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the control plane provider to be ready")
		WaitFor(ctx, For(cpProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&cpProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(cpProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(cpProvider.Status.InstalledVersion, &cpProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, cpProvider)).To(Succeed())

		By("Waiting for the control plane provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an InfrastructureProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		infraProvider := &operatorv1.InfrastructureProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      infraProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      infraProviderDeploymentName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the infrastructure provider to be ready")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&infraProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(infraProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(infraProvider.Status.InstalledVersion, &infraProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, infraProvider)).To(Succeed())

		By("Waiting for the infrastructure provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an AddonProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		addonProvider := &operatorv1.AddonProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      addonProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.AddonProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: "v0.1.0-alpha.10", // Remove to use latest when helm provider is stabilized
				},
			},
		}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      addonProviderDeploymentName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, addonProvider)).To(Succeed())

		By("Waiting for the addon provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the addon provider to be ready")
		WaitFor(ctx, For(addonProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&addonProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(addonProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(addonProvider.Status.InstalledVersion, &addonProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, addonProvider)).To(Succeed())

		By("Waiting for the addon provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete an IPAMProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		ipamProvider := &operatorv1.IPAMProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ipamProviderName,
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.IPAMProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					FetchConfig: &operatorv1.FetchConfiguration{
						URL: ipamProviderURL,
					},
				},
			},
		}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      ipamProviderDeploymentName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, ipamProvider)).To(Succeed())

		By("Waiting for the ipam provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the ipam provider to be ready")
		WaitFor(ctx, For(ipamProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&ipamProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(ipamProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(ipamProvider.Status.InstalledVersion, &ipamProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, ipamProvider)).To(Succeed())

		By("Waiting for the ipam provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade a CoreProvider (v1.7.7 -> latest)", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Get(ctx, client.ObjectKeyFromObject(coreProvider), coreProvider)).To(Succeed())

		coreProvider.Spec.Version = ""

		Expect(bootstrapCluster.Update(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: coreProviderDeploymentName, Namespace: operatorNamespace}},
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for core provider to be ready")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&coreProvider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provide status.InstalledVersion to be set")
		WaitFor(ctx, For(coreProvider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(coreProvider.Status.InstalledVersion, &coreProvider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete custom provider with OCI override", func() {
		fs, err := file.New(customManifestsFolder)
		Expect(err).ToNot(HaveOccurred())

		defer func() {
			Expect(fs.Close()).To(Succeed())
		}()

		fds := []v1.Descriptor{}

		fileDescriptor, err := fs.Add(ctx, "infrastructure-custom-v0.0.1-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-custom-v0.0.1-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.1-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.1-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		opts := oras.PackManifestOptions{
			Layers: fds,
		}

		manifestDescriptor, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, opts)
		Expect(err).ToNot(HaveOccurred())

		Expect(fs.Tag(ctx, manifestDescriptor, "v0.0.1")).ToNot(HaveOccurred())

		repo, err := remote.NewRepository("ttl.sh/cluster-api-operator-custom")
		Expect(err).ToNot(HaveOccurred())

		_, err = oras.Copy(ctx, fs, "v0.0.1", repo, "5m", oras.DefaultCopyOptions)
		Expect(err).ToNot(HaveOccurred())

		bootstrapCluster := bootstrapClusterProxy.GetClient()
		provider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom",
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: "v0.0.1",
					FetchConfig: &operatorv1.FetchConfiguration{
						OCIConfiguration: operatorv1.OCIConfiguration{
							OCI: "ttl.sh/cluster-api-operator-custom:5m",
						},
					},
				},
			},
		}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, provider)).To(Succeed())

		By("Waiting for the custom provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the custom provider to be ready")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&provider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(provider.Status.InstalledVersion, &provider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, provider)).To(Succeed())

		By("Waiting for the custom provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully create and delete docker provider with OCI override", func() {
		fs, err := file.New(customManifestsFolder)
		Expect(err).ToNot(HaveOccurred())

		defer func() {
			Expect(fs.Close()).To(Succeed())
		}()

		fds := []v1.Descriptor{}

		fileDescriptor, err := fs.Add(ctx, "infrastructure-custom-v0.0.1-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-custom-v0.0.1-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.1-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.1-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		opts := oras.PackManifestOptions{
			Layers: fds,
		}

		manifestDescriptor, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, opts)
		Expect(err).ToNot(HaveOccurred())

		Expect(fs.Tag(ctx, manifestDescriptor, "v0.0.1")).ToNot(HaveOccurred())

		repo, err := remote.NewRepository("ttl.sh/cluster-api-operator-custom")
		Expect(err).ToNot(HaveOccurred())

		_, err = oras.Copy(ctx, fs, "v0.0.1", repo, "5m", oras.DefaultCopyOptions)
		Expect(err).ToNot(HaveOccurred())

		bootstrapCluster := bootstrapClusterProxy.GetClient()
		provider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "docker",
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: "v0.0.1",
					FetchConfig: &operatorv1.FetchConfiguration{
						OCIConfiguration: operatorv1.OCIConfiguration{
							OCI: "ttl.sh/cluster-api-operator-custom:5m",
						},
					},
				},
			},
		}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, provider)).To(Succeed())

		By("Waiting for the docker provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the docker provider to be ready")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&provider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(provider.Status.InstalledVersion, &provider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, provider)).To(Succeed())

		By("Waiting for the custom docker provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully upgrade docker provider with OCI override", func() {
		fs, err := file.New(customManifestsFolder)
		Expect(err).ToNot(HaveOccurred())

		defer func() {
			Expect(fs.Close()).To(Succeed())
		}()

		fds := []v1.Descriptor{}

		fileDescriptor, err := fs.Add(ctx, "infrastructure-docker-v0.0.1-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.1-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.2-metadata.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		fileDescriptor, err = fs.Add(ctx, "infrastructure-docker-v0.0.2-components.yaml", mediaType, "")
		Expect(err).ToNot(HaveOccurred())
		fds = append(fds, fileDescriptor)

		opts := oras.PackManifestOptions{
			Layers: fds,
		}

		manifestDescriptor, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, opts)
		Expect(err).ToNot(HaveOccurred())

		Expect(fs.Tag(ctx, manifestDescriptor, "5m")).ToNot(HaveOccurred())

		repo, err := remote.NewRepository("ttl.sh/cluster-api-operator-upgrade")
		Expect(err).ToNot(HaveOccurred())

		_, err = oras.Copy(ctx, fs, "5m", repo, "5m", oras.DefaultCopyOptions)
		Expect(err).ToNot(HaveOccurred())

		bootstrapCluster := bootstrapClusterProxy.GetClient()
		provider := &operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "docker",
				Namespace: operatorNamespace,
			},
			Spec: operatorv1.InfrastructureProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					Version: "v0.0.1",
					FetchConfig: &operatorv1.FetchConfiguration{
						OCIConfiguration: operatorv1.OCIConfiguration{
							OCI: "ttl.sh/cluster-api-operator-upgrade:5m",
						},
					},
				},
			},
		}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "busybox",
			Namespace: operatorNamespace,
		}}
		Expect(bootstrapCluster.Create(ctx, provider)).To(Succeed())

		By("Waiting for the docker provider deployment to be ready")
		framework.WaitForDeploymentsAvailable(ctx, framework.WaitForDeploymentsAvailableInput{
			Getter:     bootstrapCluster,
			Deployment: deployment,
		}, e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the docker provider to be ready")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&provider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(provider.Status.InstalledVersion, &provider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Updating version to v0.0.2 to initiate upgrade")
		provider.Spec.Version = "v0.0.2"
		Expect(bootstrapCluster.Update(ctx, provider)).To(Succeed())

		By("Waiting for status.InstalledVersion to be set")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(func() bool {
			return ptr.Equal(provider.Status.InstalledVersion, &provider.Spec.Version)
		}), e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the docker provider to be ready")
		WaitFor(ctx, For(provider).In(bootstrapCluster).ToSatisfy(
			HaveStatusCondition(&provider.Status.Conditions, operatorv1.ProviderInstalledCondition)),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		Expect(bootstrapCluster.Delete(ctx, provider)).To(Succeed())

		By("Waiting for the custom docker provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})

	It("should successfully delete a CoreProvider", func() {
		bootstrapCluster := bootstrapClusterProxy.GetClient()
		coreProvider := &operatorv1.CoreProvider{ObjectMeta: metav1.ObjectMeta{
			Name:      coreProviderName,
			Namespace: operatorNamespace,
		}}
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Namespace: operatorNamespace,
			Name:      coreProviderDeploymentName,
		}}

		Expect(bootstrapCluster.Delete(ctx, coreProvider)).To(Succeed())

		By("Waiting for the core provider deployment to be deleted")
		WaitForDelete(ctx, For(deployment).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)

		By("Waiting for the core provider object to be deleted")
		WaitForDelete(ctx, For(coreProvider).In(bootstrapCluster),
			e2eConfig.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
	})
})
