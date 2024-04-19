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

package patch

import (
	"testing"

	. "github.com/onsi/gomega"
	utilyaml "sigs.k8s.io/cluster-api/util/yaml"
)

func TestApplyPatches(t *testing.T) {
	testCases := []struct {
		name                       string
		objectsToPatchYaml         string
		expectedPatchedObjectsYaml string
		patches                    []string
		expectedError              bool
	}{
		{
			name:                       "should patch objects with multiple patches",
			objectsToPatchYaml:         testObjectsToPatchYaml,
			expectedPatchedObjectsYaml: expectedTestPatchedObjectsYaml,
			patches:                    []string{addServiceAccoungPatchRBAC, addLabelPatchService, removeSelectorPatchService, addSelectorPatchService, changePortOnSecondService},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			objectToPatch, err := utilyaml.ToUnstructured([]byte(tc.objectsToPatchYaml))
			g.Expect(err).NotTo(HaveOccurred())

			result, err := ApplyPatches(objectToPatch, tc.patches)
			if tc.expectedError {
				g.Expect(err).To(HaveOccurred())
			}

			g.Expect(err).NotTo(HaveOccurred())

			resultYaml, err := utilyaml.FromUnstructured(result)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(string(resultYaml)).To(Equal(tc.expectedPatchedObjectsYaml))
		})
	}
}

const testObjectsToPatchYaml = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    some-label: value
  name: rolebinding-name
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: role-name
subjects:
- kind: ServiceAccount
  name: serviceaccount-name
  namespace: namespace-name
---
apiVersion: v1
kind: Service
metadata:
  labels:
    some-label: value
  name: service-name-1
  namespace: namespace-name
spec:
  ports:
  - port: 443
    targetPort: webhook-server
  selector:
    some-label: value
---
apiVersion: v1
kind: Service
metadata:
  labels:
    some-label: value
  name: service-name-2
  namespace: namespace-name
spec:
  ports:
  - port: 443
    targetPort: webhook-server
  selector:
    some-label: value`

const addServiceAccoungPatchRBAC = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
subjects:
- kind: ServiceAccount
  name: serviceaccount-name
  namespace: namespace-name
- kind: ServiceAccount
  name: test-service-account
  namespace: test-namespace`

const addLabelPatchService = `---
apiVersion: v1
kind: Service
metadata:
  labels:
    test-label: test-value`

const removeSelectorPatchService = `apiVersion: v1
kind: Service
spec:
  selector:`

const addSelectorPatchService = `apiVersion: v1
kind: Service
spec:
  selector:
    test-label: test-value`

const changePortOnSecondService = `---
apiVersion: v1
kind: Service
metadata:
  name: service-name-2
  namespace: namespace-name
spec:
  ports:
  - port: 7777
    targetPort: webhook-server`

const expectedTestPatchedObjectsYaml = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    some-label: value
  name: rolebinding-name
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: role-name
subjects:
- kind: ServiceAccount
  name: serviceaccount-name
  namespace: namespace-name
- kind: ServiceAccount
  name: test-service-account
  namespace: test-namespace
---
apiVersion: v1
kind: Service
metadata:
  labels:
    some-label: value
    test-label: test-value
  name: service-name-1
  namespace: namespace-name
spec:
  ports:
  - port: 443
    targetPort: webhook-server
  selector:
    test-label: test-value
---
apiVersion: v1
kind: Service
metadata:
  labels:
    some-label: value
    test-label: test-value
  name: service-name-2
  namespace: namespace-name
spec:
  ports:
  - port: 7777
    targetPort: webhook-server
  selector:
    test-label: test-value`
