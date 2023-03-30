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

package controllers

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/utils/pointer"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
	configv1 "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

func TestCustomizeDeployment(t *testing.T) {
	sevenHours, _ := time.ParseDuration("7h")
	memTestQuantity, _ := resource.ParseQuantity("16Gi")
	managerDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "manager",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "manager",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "manager",
						Image: "registry.k8s.io/a-manager:1.6.2",
						Env: []corev1.EnvVar{
							{
								Name:  "test1",
								Value: "value1",
							},
						},
						Args: []string{"--webhook-port=2345"},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.FromString("healthz"),
								},
							},
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/readyz",
									Port: intstr.FromString("healthz"),
								},
							},
						},
					}},
				},
			},
		},
	}
	tests := []struct {
		name                   string
		inputDeploymentSpec    *operatorv1.DeploymentSpec
		inputManagerSpec       *operatorv1.ManagerSpec
		expectedDeploymentSpec func(*appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool)
	}{
		{
			name:                "empty",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				if !reflect.DeepEqual(inputDS.Replicas, managerDepl.Spec.Replicas) {
					return &managerDepl.Spec, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.NodeSelector, managerDepl.Spec.Template.Spec.NodeSelector) {
					return &managerDepl.Spec, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Tolerations, managerDepl.Spec.Template.Spec.Tolerations) {
					return &managerDepl.Spec, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Affinity, managerDepl.Spec.Template.Spec.Affinity) {
					return &managerDepl.Spec, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers, managerDepl.Spec.Template.Spec.Containers) {
					return &managerDepl.Spec, false
				}

				return &managerDepl.Spec, true
			},
		},
		{
			name: "only replicas modified",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				Replicas: pointer.Int(3),
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Replicas: pointer.Int32(3),
				}

				return expectedDS, reflect.DeepEqual(inputDS.Replicas, expectedDS.Replicas)
			},
		},
		{
			name: "only node selector modified",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				NodeSelector: map[string]string{"a": "b"},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{"a": "b"},
						},
					},
				}

				return expectedDS, reflect.DeepEqual(inputDS.Template.Spec.NodeSelector, expectedDS.Template.Spec.NodeSelector)
			},
		},
		{
			name: "only tolerations modified",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				Tolerations: []corev1.Toleration{
					{
						Key:    "node-role.kubernetes.io/master",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Tolerations: []corev1.Toleration{
								{
									Key:    "node-role.kubernetes.io/master",
									Effect: corev1.TaintEffectNoSchedule,
								},
							},
						},
					},
				}

				return expectedDS, reflect.DeepEqual(inputDS.Template.Spec.Tolerations, expectedDS.Template.Spec.Tolerations)
			},
		},
		{
			name: "only affinity modified",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{},
								},
							},
						},
					},
				},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{},
											},
										},
									},
								},
							},
						},
					},
				}

				return expectedDS, reflect.DeepEqual(inputDS.Template.Spec.Affinity, expectedDS.Template.Spec.Affinity)
			},
		},
		{
			name: "only containers modified",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				Containers: []operatorv1.ContainerSpec{
					{
						Name: "manager",
						Image: &operatorv1.ImageMeta{
							Name:       "mydns",
							Repository: "quay.io/dev",
							Tag:        "v3.4.2",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "test1",
								Value: "value2",
							},
							{
								Name:  "new1",
								Value: "value22",
							},
						},
						Args: map[string]string{
							"--webhook-port": "3456",
							"--log_dir":      "/var/log",
						},
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceMemory: memTestQuantity},
						},
					},
				},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "manager",
									Image: "quay.io/dev/mydns:v3.4.2",
									Env: []corev1.EnvVar{
										{
											Name:  "test1",
											Value: "value2",
										},
										{
											Name:  "new1",
											Value: "value22",
										},
									},
									Args: []string{
										"--webhook-port=3456",
										"--log_dir=/var/log",
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{corev1.ResourceMemory: memTestQuantity},
									},
								},
							},
						},
					},
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Name, expectedDS.Template.Spec.Containers[0].Name) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Image, expectedDS.Template.Spec.Containers[0].Image) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Env, expectedDS.Template.Spec.Containers[0].Env) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Args, expectedDS.Template.Spec.Containers[0].Args) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Resources, expectedDS.Template.Spec.Containers[0].Resources) {
					return expectedDS, false
				}

				return expectedDS, true
			},
		},
		{
			name: "all deployment options",
			inputDeploymentSpec: &operatorv1.DeploymentSpec{
				Replicas:     pointer.Int(3),
				NodeSelector: map[string]string{"a": "b"},
				Tolerations: []corev1.Toleration{
					{
						Key:    "node-role.kubernetes.io/master",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{},
								},
							},
						},
					},
				},
				Containers: []operatorv1.ContainerSpec{
					{
						Name: "manager",
						Image: &operatorv1.ImageMeta{
							Name:       "mydns",
							Repository: "quay.io/dev",
							Tag:        "v3.4.2",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "test1",
								Value: "value2",
							},
							{
								Name:  "new1",
								Value: "value22",
							},
						},
						Args: map[string]string{
							"--webhook-port": "3456",
							"--log_dir":      "/var/log",
						},
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceMemory: memTestQuantity},
						},
						Command: []string{"/expected"},
					},
				},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Replicas: pointer.Int32(3),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{"a": "b"},
							Tolerations: []corev1.Toleration{
								{
									Key:    "node-role.kubernetes.io/master",
									Effect: corev1.TaintEffectNoSchedule,
								},
							},
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{},
											},
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "manager",
									Image: "quay.io/dev/mydns:v3.4.2",
									Env: []corev1.EnvVar{
										{
											Name:  "test1",
											Value: "value2",
										},
										{
											Name:  "new1",
											Value: "value22",
										},
									},
									Args: []string{
										"--webhook-port=3456",
										"--log_dir=/var/log",
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{corev1.ResourceMemory: memTestQuantity},
									},
									Command: []string{"/expected"},
								},
							},
						},
					},
				}

				if !reflect.DeepEqual(inputDS.Replicas, expectedDS.Replicas) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.NodeSelector, expectedDS.Template.Spec.NodeSelector) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Tolerations, expectedDS.Template.Spec.Tolerations) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Affinity, expectedDS.Template.Spec.Affinity) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Name, expectedDS.Template.Spec.Containers[0].Name) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Image, expectedDS.Template.Spec.Containers[0].Image) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Env, expectedDS.Template.Spec.Containers[0].Env) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Args, expectedDS.Template.Spec.Containers[0].Args) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Resources, expectedDS.Template.Spec.Containers[0].Resources) {
					return expectedDS, false
				}
				if !reflect.DeepEqual(inputDS.Template.Spec.Containers[0].Command, expectedDS.Template.Spec.Containers[0].Command) {
					return expectedDS, false
				}

				return expectedDS, true
			},
		},
		{
			name: "all manager options",
			inputManagerSpec: &operatorv1.ManagerSpec{
				FeatureGates:    map[string]bool{"TEST": true, "ANOTHER": false},
				ProfilerAddress: "localhost:1234",
				Verbosity:       5,
				ControllerManagerConfigurationSpec: configv1.ControllerManagerConfigurationSpec{
					CacheNamespace: "testNS",
					SyncPeriod:     &metav1.Duration{Duration: sevenHours},
					Controller:     &configv1.ControllerConfigurationSpec{GroupKindConcurrency: map[string]int{"machine": 3}},
					Metrics:        configv1.ControllerMetrics{BindAddress: ":4567"},
					Health: configv1.ControllerHealth{
						HealthProbeBindAddress: ":6789",
						ReadinessEndpointName:  "readyish",
						LivenessEndpointName:   "mostly",
					},
					Webhook: configv1.ControllerWebhook{
						Port:    pointer.Int(3579),
						CertDir: "/tmp/certs",
					},
					LeaderElection: &configv1alpha1.LeaderElectionConfiguration{
						LeaderElect:       pointer.Bool(true),
						ResourceName:      "foo",
						ResourceNamespace: "here",
						LeaseDuration:     metav1.Duration{Duration: sevenHours},
						RenewDeadline:     metav1.Duration{Duration: sevenHours},
						RetryPeriod:       metav1.Duration{Duration: sevenHours},
					},
				},
			},
			expectedDeploymentSpec: func(inputDS *appsv1.DeploymentSpec) (*appsv1.DeploymentSpec, bool) {
				expectedDS := &appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name: "manager",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "manager",
									Image: "registry.k8s.io/a-manager:1.6.2",
									Env: []corev1.EnvVar{
										{
											Name:  "test1",
											Value: "value1",
										},
									},
									Args: []string{
										"--webhook-port=3579",
										"--machine-concurrency=3",
										"--namespace=testNS",
										"--health-addr=:6789",
										"--enable-leader-election=true",
										"--leader-election-id=here/foo",
										"--leader-elect-lease-duration=25200s",
										"--leader-elect-renew-deadline=25200s",
										"--leader-elect-retry-period=25200s",
										"--metrics-addr=:4567",
										"--webhook-cert-dir=/tmp/certs",
										"--sync-period=25200s",
										"--profiler-address=localhost:1234",
										"--v=5",
										"--feature-gates=ANOTHER=false,TEST=true"},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/mostly",
												Port: intstr.FromString("healthz"),
											},
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											HTTPGet: &corev1.HTTPGetAction{
												Path: "/readyish",
												Port: intstr.FromString("healthz"),
											},
										},
									},
								},
							},
						},
					},
				}

				return expectedDS, reflect.DeepEqual(inputDS.Template.Spec.Containers[0], expectedDS.Template.Spec.Containers[0])
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deployment := managerDepl.DeepCopy()
			customizeDeployment(operatorv1.ProviderSpec{
				Deployment: tc.inputDeploymentSpec,
				Manager:    tc.inputManagerSpec,
			}, deployment)

			if ds, expected := tc.expectedDeploymentSpec(&deployment.Spec); !expected {
				t.Error(cmp.Diff(ds, deployment.Spec))
			}
		})
	}
}
