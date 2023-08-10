//go:build e2e
// +build e2e

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

package framework

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"

	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func HaveStatusCondition(conditions *clusterv1.Conditions, condition clusterv1.ConditionType) Condition {
	return func() bool {
		By(fmt.Sprintf("Checking if %s condition is set...", condition))
		for _, c := range *conditions {
			if c.Type == condition && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
}
