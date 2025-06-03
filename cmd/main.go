/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	goruntime "runtime"
	"time"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	"sigs.k8s.io/cluster-api-operator/internal/webhook"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/flags"
	"sigs.k8s.io/cluster-api/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha2"
	providercontroller "sigs.k8s.io/cluster-api-operator/internal/controller"
	healtchcheckcontroller "sigs.k8s.io/cluster-api-operator/internal/controller/healthcheck"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// flags.
	enableLeaderElection        bool
	leaderElectionLeaseDuration time.Duration
	leaderElectionRenewDeadline time.Duration
	leaderElectionRetryPeriod   time.Duration
	watchFilterValue            string
	watchNamespace              string
	profilerAddress             string
	enableContentionProfiling   bool
	concurrencyNumber           int
	managerConcurrency          int
	syncPeriod                  time.Duration
	webhookPort                 int
	webhookCertDir              string
	healthAddr                  string
	watchConfigSecretChanges    bool
	managerOptions              = flags.ManagerOptions{}
)

func init() {
	klog.InitFlags(nil)

	// +kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme))
}

// InitFlags initializes the flags.
func InitFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	fs.DurationVar(&leaderElectionLeaseDuration, "leader-elect-lease-duration", 15*time.Second,
		"Interval at which non-leader candidates will wait to force acquire leadership (duration string)")

	fs.DurationVar(&leaderElectionRenewDeadline, "leader-elect-renew-deadline", 10*time.Second,
		"Duration that the leading controller manager will retry refreshing leadership before giving up (duration string)")

	fs.DurationVar(&leaderElectionRetryPeriod, "leader-elect-retry-period", 2*time.Second,
		"Duration the LeaderElector clients should wait between tries of actions (duration string)")

	fs.StringVar(&watchFilterValue, "watch-filter", "",
		fmt.Sprintf("Label value that the controller watches to reconcile cluster-api objects. Label key is always %s. If unspecified, the controller watches for all cluster-api objects.", clusterv1.WatchLabel))

	fs.BoolVar(&watchConfigSecretChanges, "watch-configsecret", false,
		"Watch for changes to the ConfigSecret resource and reconcile all providers using it.")

	fs.StringVar(&watchNamespace, "namespace", "",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.")

	fs.StringVar(&profilerAddress, "profiler-address", "localhost:6060",
		"Bind address to expose the pprof profiler (e.g. localhost:6060). Set to empty string to disable.")

	fs.BoolVar(&enableContentionProfiling, "contention-profiling", false,
		"Enable block profiling when profiler is active")

	fs.IntVar(&concurrencyNumber, "concurrency", 1,
		"Number of core resources to process simultaneously")

	fs.IntVar(&managerConcurrency, "manager-concurrency", 10,
		"Number of concurrent reconciles to process simultaneously across all controllers")

	fs.DurationVar(&syncPeriod, "sync-period", 10*time.Minute,
		"The minimum interval at which watched resources are reconciled (e.g. 15m)")

	fs.IntVar(&webhookPort, "webhook-port", 9443, "Webhook Server port")

	fs.StringVar(&webhookCertDir, "webhook-cert-dir", "/tmp/k8s-webhook-server/serving-certs/",
		"Webhook cert dir, only used when webhook-port is specified.")

	fs.StringVar(&healthAddr, "health-addr", ":9440",
		"The address the health endpoint binds to.")

	flags.AddManagerOptions(fs, &managerOptions)
}

func main() {
	InitFlags(pflag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(textlogger.NewLogger(textlogger.NewConfig()))
	restConfig := ctrl.GetConfigOrDie()

	tlsOptions, metricsOptions, err := flags.GetManagerOptions(managerOptions)
	if err != nil {
		setupLog.Error(err, "Unable to start manager: invalid flags")
		os.Exit(1)
	}

	var watchNamespaces map[string]cache.Config
	if watchNamespace != "" {
		watchNamespaces = map[string]cache.Config{
			watchNamespace: {},
		}
	}

	if enableContentionProfiling && profilerAddress != "" {
		goruntime.SetBlockProfileRate(1)
	}

	ctrlOptions := ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "controller-leader-election-capi-operator",
		LeaseDuration:          &leaderElectionLeaseDuration,
		RenewDeadline:          &leaderElectionRenewDeadline,
		RetryPeriod:            &leaderElectionRetryPeriod,
		HealthProbeBindAddress: healthAddr,
		PprofBindAddress:       profilerAddress,
		Metrics:                *metricsOptions,
		Cache: cache.Options{
			DefaultNamespaces: watchNamespaces,
			SyncPeriod:        &syncPeriod,
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.ConfigMap{},
					&corev1.Secret{},
				},
			},
		},
		Controller: config.Controller{
			MaxConcurrentReconciles: managerConcurrency,
		},
		WebhookServer: ctrlwebhook.NewServer(
			ctrlwebhook.Options{
				Port:    webhookPort,
				CertDir: webhookCertDir,
				TLSOpts: tlsOptions,
			},
		),
	}

	mgr, err := ctrl.NewManager(restConfig, ctrlOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup the context that's going to be used in controllers and for the manager.
	ctx := ctrl.SetupSignalHandler()

	setupChecks(mgr)
	setupReconcilers(ctx, mgr, watchConfigSecretChanges)
	setupWebhooks(mgr)

	// +kubebuilder:scaffold:builder
	setupLog.Info("starting manager", "version", version.Get().String())

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupChecks(mgr ctrl.Manager) {
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create ready check")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to create health check")
		os.Exit(1)
	}
}

func setupReconcilers(ctx context.Context, mgr ctrl.Manager, watchConfigSecretChanges bool) {
	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.CoreProvider{},
		ProviderList:             &operatorv1.CoreProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CoreProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.InfrastructureProvider{},
		ProviderList:             &operatorv1.InfrastructureProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InfrastructureProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.BootstrapProvider{},
		ProviderList:             &operatorv1.BootstrapProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BootstrapProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.ControlPlaneProvider{},
		ProviderList:             &operatorv1.ControlPlaneProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ControlPlaneProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.AddonProvider{},
		ProviderList:             &operatorv1.AddonProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AddonProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.IPAMProvider{},
		ProviderList:             &operatorv1.IPAMProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IPAMProvider")
		os.Exit(1)
	}

	if err := (&providercontroller.GenericProviderReconciler{
		Provider:                 &operatorv1.RuntimeExtensionProvider{},
		ProviderList:             &operatorv1.RuntimeExtensionProviderList{},
		Client:                   mgr.GetClient(),
		Config:                   mgr.GetConfig(),
		WatchConfigSecretChanges: watchConfigSecretChanges,
		WatchCoreProviderChanges: true,
	}).SetupWithManager(ctx, mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RuntimeExtensionProvider")
		os.Exit(1)
	}

	if err := (&healtchcheckcontroller.ProviderHealthCheckReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr, concurrency(concurrencyNumber)); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Healthcheck")
		os.Exit(1)
	}
}

func setupWebhooks(mgr ctrl.Manager) {
	if err := (&webhook.CoreProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CoreProvider")
		os.Exit(1)
	}

	if err := (&webhook.BootstrapProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "BootstrapProvider")
		os.Exit(1)
	}

	if err := (&webhook.ControlPlaneProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ControlPlaneProvider")
		os.Exit(1)
	}

	if err := (&webhook.InfrastructureProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "InfrastructureProvider")
		os.Exit(1)
	}

	if err := (&webhook.AddonProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AddonProvider")
		os.Exit(1)
	}

	if err := (&webhook.IPAMProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "IPAMProvider")
		os.Exit(1)
	}

	if err := (&webhook.RuntimeExtensionProviderWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "RuntimeExtensionProvider")
		os.Exit(1)
	}
}

func concurrency(c int) controller.Options {
	return controller.Options{MaxConcurrentReconciles: c}
}
