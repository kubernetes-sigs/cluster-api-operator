package controller

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	"sigs.k8s.io/cluster-api-operator/internal/controller/genericprovider"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestSecretReconciler(t *testing.T) {
	ctx := ctx

	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(v1.AddToScheme(scheme)).To(Succeed())
	g.Expect(operatorv1.AddToScheme(scheme)).To(Succeed())

	providersUsingTheSecret := []client.Object{
		&operatorv1.AddonProvider{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "some-addon",
			},
			Spec: operatorv1.AddonProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					ConfigSecret: &operatorv1.SecretReference{
						Name: "secret-in-default-namespace",
					},
				},
			},
		},
		&operatorv1.ControlPlaneProvider{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "some-namespace",
				Name:      "some-control-plane",
			},
			Spec: operatorv1.ControlPlaneProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					ConfigSecret: &operatorv1.SecretReference{
						Name:      "secret-in-default-namespace",
						Namespace: "default",
					},
				},
			},
		},
	}
	providersNotUsingTheSecret := []client.Object{
		&operatorv1.InfrastructureProvider{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "some-namespace",
				Name:      "some-infra",
			},
		},
		&operatorv1.IPAMProvider{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "some-addon",
			},
			Spec: operatorv1.IPAMProviderSpec{
				ProviderSpec: operatorv1.ProviderSpec{
					ConfigSecret: &operatorv1.SecretReference{
						Name: "other-secret-name",
					},
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(providersUsingTheSecret...).WithObjects(providersNotUsingTheSecret...).Build()

	r := &SecretReconciler{
		Client: k8sClient,
		ProviderLists: []genericprovider.GenericProviderList{
			&operatorv1.AddonProviderList{},
			&operatorv1.ControlPlaneProviderList{},
			&operatorv1.InfrastructureProviderList{},
			&operatorv1.IPAMProviderList{},
			&operatorv1.BootstrapProviderList{},
		},
	}

	t.Run("When the secret does not exist", func(t *testing.T) {
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "secret-in-default-namespace", Namespace: "default"}})
		g.Expect(err).ToNot(HaveOccurred())

		assertProvidersAreAnnotatedWithSecretVersions(g, ctx, k8sClient, providersUsingTheSecret, providersNotUsingTheSecret, "")

		t.Run("Any subsequent reconciliation is successful", func(t *testing.T) {
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "secret-in-default-namespace", Namespace: "default"}})
			g.Expect(err).ToNot(HaveOccurred())

			assertProvidersAreAnnotatedWithSecretVersions(g, ctx, k8sClient, providersUsingTheSecret, providersNotUsingTheSecret, "")
		})
	})

	t.Run("When the secret exists", func(t *testing.T) {
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "secret-in-default-namespace",
			},
			Data: map[string][]byte{
				"key": []byte("value"),
			},
		}
		g.Expect(r.Client.Create(ctx, secret)).To(Succeed())

		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "secret-in-default-namespace", Namespace: "default"}})
		g.Expect(err).ToNot(HaveOccurred())
		assertProvidersAreAnnotatedWithSecretVersions(g, ctx, k8sClient, providersUsingTheSecret, providersNotUsingTheSecret, "fed7a27106a07691449b9cf7f57536328004b134d358d8fcafaf6a2a06f99d50")

		t.Run("Any subsequent reconciliation is successful", func(t *testing.T) {
			_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: "secret-in-default-namespace", Namespace: "default"}})
			g.Expect(err).ToNot(HaveOccurred())
		})
		assertProvidersAreAnnotatedWithSecretVersions(g, ctx, k8sClient, providersUsingTheSecret, providersNotUsingTheSecret, "fed7a27106a07691449b9cf7f57536328004b134d358d8fcafaf6a2a06f99d50")
	})
}

func assertProvidersAreAnnotatedWithSecretVersions(g *WithT, ctx context.Context, k8sClient client.Client, providersUsingTheSecret, providersNotUsingTheSecret []client.Object, secretHash string) {

	for _, provider := range providersUsingTheSecret {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		g.Expect(provider.GetAnnotations()[observedSpecHashAnnotation]).To(Equal(secretHash))
	}
	for _, provider := range providersNotUsingTheSecret {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(provider), provider)).To(Succeed())
		g.Expect(provider.GetAnnotations()).NotTo(HaveKey(observedSpecHashAnnotation))
	}
}
