/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

// inspectImages identifies the container images required to install the objects defined in the objs.
// NB. The implemented approach is specific for the provider components YAML & for the cert-manager manifest; it is not
// intended to cover all the possible objects used to deploy containers existing in Kubernetes.
func inspectImages(objs []unstructured.Unstructured) ([]string, error) {
	images := []string{}

	for i := range objs {
		o := objs[i]

		var podSpec corev1.PodSpec

		switch o.GetKind() {
		case deploymentKind:
			d := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(&o, d, nil); err != nil {
				return nil, err
			}

			podSpec = d.Spec.Template.Spec
		case daemonSetKind:
			d := &appsv1.DaemonSet{}
			if err := scheme.Scheme.Convert(&o, d, nil); err != nil {
				return nil, err
			}

			podSpec = d.Spec.Template.Spec
		default:
			continue
		}

		for _, c := range podSpec.Containers {
			images = append(images, c.Image)
		}

		for _, c := range podSpec.InitContainers {
			images = append(images, c.Image)
		}
	}

	return images, nil
}

func TestFixImages(t *testing.T) {
	type args struct {
		objs           []unstructured.Unstructured
		alterImageFunc func(image string) (string, error)
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "fix deployment containers images",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       deploymentKind,
							"spec": map[string]interface{}{
								"template": map[string]interface{}{
									"spec": map[string]interface{}{
										"containers": []map[string]interface{}{
											{
												"image": "container-image",
											},
										},
										"initContainers": []map[string]interface{}{
											{
												"image": "init-container-image",
											},
										},
									},
								},
							},
						},
					},
				},
				alterImageFunc: func(image string) (string, error) {
					return fmt.Sprintf("foo-%s", image), nil
				},
			},
			want:    []string{"foo-container-image", "foo-init-container-image"},
			wantErr: false,
		},
		{
			name: "fix daemonSet containers images",
			args: args{
				objs: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       daemonSetKind,
							"spec": map[string]interface{}{
								"template": map[string]interface{}{
									"spec": map[string]interface{}{
										"containers": []map[string]interface{}{
											{
												"image": "container-image",
											},
										},
										"initContainers": []map[string]interface{}{
											{
												"image": "init-container-image",
											},
										},
									},
								},
							},
						},
					},
				},
				alterImageFunc: func(image string) (string, error) {
					return fmt.Sprintf("foo-%s", image), nil
				},
			},
			want:    []string{"foo-container-image", "foo-init-container-image"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			got, err := fixImages(tt.args.objs, tt.args.alterImageFunc)
			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).ToNot(HaveOccurred())

			gotImages, err := inspectImages(got)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(gotImages).To(Equal(tt.want))
		})
	}
}

// mockImageMetaClient is a test double for configclient.ImageMetaClient.
type mockImageMetaClient struct {
	alterFunc func(component, image string) (string, error)
}

func (m *mockImageMetaClient) AlterImage(component, image string) (string, error) {
	return m.alterFunc(component, image)
}

func TestAlterImage(t *testing.T) {
	tests := []struct {
		name      string
		component string
		image     string
		mockFunc  func(component, image string) (string, error)
		want      string
		wantErr   bool
	}{
		{
			name:      "canonical image with override applies override",
			component: "cluster-api",
			image:     "example.com/controller:v1.0.0",
			mockFunc: func(component, image string) (string, error) {
				return "example.com/custom:v2.0.0", nil
			},
			want:    "example.com/custom:v2.0.0",
			wantErr: false,
		},
		{
			name:      "non-canonical image returns original on canonical error",
			component: "cluster-api",
			image:     "example.com/controller:v1.0.0",
			mockFunc: func(component, image string) (string, error) {
				return "", fmt.Errorf("couldn't parse image name: repository name must be canonical")
			},
			want:    "example.com/controller:v1.0.0",
			wantErr: false,
		},
		{
			name:      "other errors are propagated",
			component: "cluster-api",
			image:     "example.com/controller:v1.0.0",
			mockFunc: func(component, image string) (string, error) {
				return "", fmt.Errorf("test")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			mock := &mockImageMetaClient{alterFunc: tt.mockFunc}
			result, err := alterImage(tt.component, tt.image, mock)

			if tt.wantErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(result).To(Equal(tt.want))
		})
	}
}

func TestIsCanonicalError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "canonical error with 'repository name must be canonical'",
			err:  fmt.Errorf("repository name must be canonical"),
			want: true,
		},
		{
			name: "canonical error with 'couldn't parse image name'",
			err:  fmt.Errorf("couldn't parse image name: invalid format"),
			want: true,
		},
		{
			name: "other error returns false",
			err:  fmt.Errorf("test"),
			want: false,
		},
		{
			name: "empty error message returns false",
			err:  fmt.Errorf(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(isCanonicalError(tt.err)).To(Equal(tt.want))
		})
	}
}
