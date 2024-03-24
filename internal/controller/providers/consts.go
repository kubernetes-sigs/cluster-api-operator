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

package providers

import (
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api-operator/internal/controller/generic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// preflightFailedRequeueAfter is how long to wait before trying to reconcile
	// if some preflight check has failed.
	preflightFailedRequeueAfter = 30 * time.Second

	incorrectCoreProviderNameMessage             = "Incorrect CoreProvider name: %s. It should be %s"
	moreThanOneCoreProviderInstanceExistsMessage = "CoreProvider already exists in the cluster. Only one is allowed."
	waitingForCoreProviderReadyMessage           = "Waiting for the core provider to be installed."
)

type ConnectorStub struct{}

// GetClient implements generic.Connector.
func (c ConnectorStub) GetClient() client.Client {
	return nil
}

// GetConfig implements generic.Connector.
func (c ConnectorStub) GetConfig() *rest.Config {
	return nil
}

func init() {
	generic.ProviderReconcilers[(&CoreProviderReconciler{}).ClusterctlProviderType()] = NewCoreProviderReconcier(ConnectorStub{})
	generic.ProviderReconcilers[(&InfrastructureProviderReconciler{}).ClusterctlProviderType()] = NewInfrastructureProviderReconciler(ConnectorStub{})
	generic.ProviderReconcilers[(&BootstrapProviderReconciler{}).ClusterctlProviderType()] = NewBootstrapProviderReconciler(ConnectorStub{})
	generic.ProviderReconcilers[(&ControlPlaneProviderReconciler{}).ClusterctlProviderType()] = NewControlPlaneProviderReconciler(ConnectorStub{})
	generic.ProviderReconcilers[(&AddonProviderReconciler{}).ClusterctlProviderType()] = NewAddonProviderReconciler(ConnectorStub{})
	generic.ProviderReconcilers[(&IPAMProviderReconciler{}).ClusterctlProviderType()] = NewIPAMProviderReconciler(ConnectorStub{})
}
