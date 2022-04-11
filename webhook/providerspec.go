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

package webhook

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha1"
)

func validateProviderSpec(spec operatorv1.ProviderSpec) field.ErrorList {
	var allErrs field.ErrorList

	if spec.Version != "" {
		if _, err := versionutil.ParseSemantic(spec.Version); err != nil {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec").Child("version"),
					spec.Version,
					fmt.Sprintf("invalid semantic version: %v", err)))
		}
	}

	if spec.FetchConfig != nil && spec.FetchConfig.Selector != nil && spec.FetchConfig.Selector.MatchLabels != nil {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec").Child("fetchConfig"),
				spec.Version,
				"can't use selector and matchlabels, only one option is allowed"))
	}

	return allErrs
}
