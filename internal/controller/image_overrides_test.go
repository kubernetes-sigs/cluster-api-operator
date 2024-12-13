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
