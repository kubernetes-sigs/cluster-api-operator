//go:build e2e

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

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func HaveStatusConditionsTrue(getter capiconditions.Getter, conditions ...clusterv1.ConditionType) Condition {
	return func() bool {
		if len(conditions) == 0 {
			By("Empty condition list provided. Can't be validated...")

			return false
		}

		for _, condition := range conditions {
			By(fmt.Sprintf("Checking if %s condition is set...", condition))

			if !capiconditions.IsTrue(getter, condition) {
				return false
			}
		}

		return true
	}
}
