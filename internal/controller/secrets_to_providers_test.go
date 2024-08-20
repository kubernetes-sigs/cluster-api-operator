package controller

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProviderSecretMapper(t *testing.T) {
	g := NewWithT(t)

	configSecretNamespace := "test-namespace"
	otherInfraProviderNamespace := "other-namespace"
	configSecretName := "infra-provider-config"

	k8sClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-using-secret",
					Namespace: configSecretNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name: configSecretName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-using-secret-from-other-namespace",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name:      configSecretName,
							Namespace: configSecretNamespace,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-infra-provider-using-secret-from-other-namespace",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name:      configSecretName,
							Namespace: configSecretNamespace,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-with-other-secret",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: &operatorv1.SecretReference{
							Name: configSecretName,
						},
					},
				},
			},
			&operatorv1.InfrastructureProvider{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "infra-provider-without-config-secret",
					Namespace: otherInfraProviderNamespace,
				},
				Spec: operatorv1.InfrastructureProviderSpec{
					ProviderSpec: operatorv1.ProviderSpec{
						ConfigSecret: nil,
					},
				},
			},
		).
		Build()

	requests := newSecretToProviderFuncMapForProviderList(k8sClient, &operatorv1.InfrastructureProviderList{})(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configSecretName,
			Namespace: configSecretNamespace,
		},
	})

	g.Expect(requests).To(HaveLen(3))
	g.Expect(requests).To(ContainElements(
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: configSecretNamespace, Name: "infra-provider-using-secret"}},
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: otherInfraProviderNamespace, Name: "infra-provider-using-secret-from-other-namespace"}},
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: otherInfraProviderNamespace, Name: "other-infra-provider-using-secret-from-other-namespace"}},
	))

}
