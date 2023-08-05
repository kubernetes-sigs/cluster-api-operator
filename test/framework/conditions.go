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
