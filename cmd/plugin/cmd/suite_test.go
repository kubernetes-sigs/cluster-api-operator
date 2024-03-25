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

package cmd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"sigs.k8s.io/cluster-api-operator/internal/envtest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"

	// We need to initialize all registered providers.
	_ "sigs.k8s.io/cluster-api-operator/internal/controller/providers"
)

const (
	waitShort = time.Second * 5
	waitLong  = time.Second * 20
)

var (
	env *envtest.Environment
	ctx = ctrl.SetupSignalHandler()
)

func TestMain(m *testing.M) {
	fmt.Println("Creating new test environment")

	env = envtest.New()

	if err := env.Manager.GetCache().IndexField(ctx, &operatorv1.AddonProvider{},
		"metadata.name",
		func(obj client.Object) []string {
			return []string{obj.GetName()}
		},
	); err != nil {
		panic(fmt.Sprintf("Error setting up name index field: %v", err))
	}

	if err := env.Manager.GetCache().IndexField(ctx, &operatorv1.AddonProvider{},
		"metadata.namespace",
		func(obj client.Object) []string {
			return []string{obj.GetNamespace()}
		},
	); err != nil {
		panic(fmt.Sprintf("Error setting up namespace index field: %v", err))
	}

	go func() {
		if err := env.Start(ctx); err != nil {
			panic(fmt.Sprintf("Failed to start the envtest manager: %v", err))
		}
	}()
	<-env.Manager.Elected()

	// Run tests
	code := m.Run()
	// Tearing down the test environment
	if err := env.Stop(); err != nil {
		panic(fmt.Sprintf("Failed to stop the envtest: %v", err))
	}

	// Report exit code
	os.Exit(code)
}
