//go:build e2e
// +build e2e

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
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	"sigs.k8s.io/cluster-api-operator/controllers/genericprovider"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ctx = context.Background()
)

const (
	timeout            = 10 * time.Minute
	operatorNamespace  = "capi-operator-system"
	providerSecretName = "provider-secret"

	coreProviderName           = "cluster-api"
	coreProviderDeploymentName = "capi-controller-manager"
	coreProviderVersion        = "v1.1.1"

	infrastructureProviderName           = "aws"
	infrastructureProviderDeploymentName = "capa-controller-manager"
	infraProviderVersion                 = "v1.4.0"

	boostrapProviderName           = "kubeadm"
	boostrapProviderDeploymentName = "capi-kubeadm-bootstrap-controller-manager"

	controlPlaneProviderName           = "kubeadm"
	controlPlaneProviderDeploymentName = "capi-kubeadm-control-plane-controller-manager"
)

func newGenericProvider(obj client.Object) (genericprovider.GenericProvider, error) {
	switch obj := obj.(type) {
	case *operatorv1.CoreProvider:
		return &genericprovider.CoreProviderWrapper{CoreProvider: obj}, nil
	case *operatorv1.BootstrapProvider:
		return &genericprovider.BootstrapProviderWrapper{BootstrapProvider: obj}, nil
	case *operatorv1.ControlPlaneProvider:
		return &genericprovider.ControlPlaneProviderWrapper{ControlPlaneProvider: obj}, nil
	case *operatorv1.InfrastructureProvider:
		return &genericprovider.InfrastructureProviderWrapper{InfrastructureProvider: obj}, nil
	default:
		providerKind := reflect.Indirect(reflect.ValueOf(obj)).Type().Name()
		failedToCastInterfaceErr := fmt.Errorf("failed to cast interface for type: %s", providerKind)
		return nil, failedToCastInterfaceErr
	}
}

func waitForProviderCondition(provider genericprovider.GenericProvider, condition clusterv1.Condition, k8sClient client.Client) {
	Eventually(func() bool {
		key := client.ObjectKey{Namespace: provider.GetObject().GetNamespace(), Name: provider.GetObject().GetName()}
		if err := k8sClient.Get(ctx, key, provider.GetObject()); err != nil {
			return false
		}

		for _, c := range provider.GetStatus().Conditions {
			if c.Type == condition.Type && c.Status == condition.Status && c.Reason == condition.Reason {
				return true
			}
		}
		return false
	}, timeout).Should(Equal(true))
}

func waitForDeploymentReady(namespace, name string, k8sClient client.Client) {
	Eventually(func() bool {
		deployment := &appsv1.Deployment{}
		key := client.ObjectKey{Namespace: namespace, Name: name}
		if err := k8sClient.Get(ctx, key, deployment); err != nil {
			return false
		}

		for _, c := range deployment.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true
			}
		}

		return false

	}, timeout).Should(Equal(true))
}

func waitForDeploymentDeleted(namespace, name string, k8sClient client.Client) {
	Eventually(func() bool {
		deployment := &appsv1.Deployment{}
		key := client.ObjectKey{Namespace: namespace, Name: name}
		if err := k8sClient.Get(ctx, key, deployment); err != nil {
			return apierrors.IsNotFound(err)
		}

		return false
	}, timeout).Should(Equal(true))
}

func waitForInstalledVersionSet(provider genericprovider.GenericProvider, version string, k8sClient client.Client) {
	Eventually(func() bool {
		key := client.ObjectKey{Namespace: operatorNamespace, Name: provider.GetName()}
		if err := k8sClient.Get(ctx, key, provider.GetObject()); err != nil {
			return false
		}

		if provider.GetStatus().InstalledVersion != nil && *provider.GetStatus().InstalledVersion == version {
			return true
		}
		return false
	}, timeout).Should(Equal(true))
}

func cleanupAndWait(ctx context.Context, cl client.Client, objs ...client.Object) error {
	if err := cleanup(ctx, cl, objs...); err != nil {
		return err
	}
	// Makes sure the cache is updated with the deleted object
	errs := []error{}
	for _, o := range objs {
		oCopy := o.DeepCopyObject().(client.Object)
		key := client.ObjectKeyFromObject(o)
		err := wait.ExponentialBackoff(
			wait.Backoff{
				Duration: 150 * time.Millisecond,
				Factor:   1.5,
				Steps:    8,
				Jitter:   0.4,
			},
			func() (done bool, err error) {
				if err := cl.Get(ctx, key, oCopy); err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					}
					return false, err
				}
				return false, nil
			})
		errs = append(errs, errors.Wrapf(err, "key %s, %s is not being deleted from the client cache", o.GetObjectKind().GroupVersionKind().String(), key))
	}
	return kerrors.NewAggregate(errs)
}

func cleanup(ctx context.Context, cl client.Client, objs ...client.Object) error {
	errs := []error{}
	for _, o := range objs {
		oCopy := o.DeepCopyObject().(client.Object)
		key := client.ObjectKeyFromObject(oCopy)

		err := cl.Get(ctx, key, oCopy)
		if apierrors.IsNotFound(err) {
			continue
		}
		errs = append(errs, err)

		// Remove finalizers from the object
		if oCopy.GetFinalizers() != nil {
			oCopy.SetFinalizers(nil)
		}

		err = cl.Update(ctx, oCopy)
		if apierrors.IsNotFound(err) {
			continue
		}
		errs = append(errs, err)

		err = cl.Delete(ctx, oCopy)
		if apierrors.IsNotFound(err) {
			continue
		}
		errs = append(errs, err)
	}
	return kerrors.NewAggregate(errs)
}
