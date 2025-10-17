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

package envtest

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"

	"sigs.k8s.io/cluster-api/util/kubeconfig"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func init() {
	klog.InitFlags(nil)

	ctrl.SetLogger(textlogger.NewLogger(textlogger.NewConfig()))

	// Calculate the scheme.
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterctlv1.AddToScheme(scheme.Scheme))
}

var (
	cacheSyncBackoff = wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Steps:    8,
		Jitter:   0.4,
	}

	errAlreadyStarted      = fmt.Errorf("environment has already been started")
	errAlreadyStopped      = fmt.Errorf("environment has already been stopped")
	clusterAPIVersionRegex = regexp.MustCompile(`^(\W)sigs.k8s.io/cluster-api v(.+)`)
)

// Environment encapsulates a Kubernetes local test environment.
type Environment struct {
	manager.Manager
	client.Client
	Config *rest.Config

	env           *envtest.Environment
	startOnce     sync.Once
	stopOnce      sync.Once
	cancelManager context.CancelFunc
}

// New creates a new environment spinning up a local api-server.
//
// This function should be called only once for each package you're running tests within,
// usually the environment is initialized in a suite_test.go file within a `BeforeSuite` ginkgo block.
func New(uncachedObjs ...client.Object) *Environment {
	// Get the root of the current file to use in CRD paths.
	_, filename, _, _ := goruntime.Caller(0)
	root := path.Join(path.Dir(filename), "..", "..")
	crdPaths := []string{
		filepath.Join(root, "config", "crd", "bases"),
		// cert-manager CRDs are stored there.
		filepath.Join(root, "test", "testdata"),
	}

	if capiPath := getFilePathToClusterctlCRDs(root); capiPath != "" {
		crdPaths = append(crdPaths, capiPath)
	}

	if err := operatorv1.AddToScheme(scheme.Scheme); err != nil {
		klog.Fatalf("Failed to set scheme for manager: %v", err)
	}

	// Create the test environment.
	env := &envtest.Environment{
		Scheme:                scheme.Scheme,
		ErrorIfCRDPathMissing: true,
		// CRDInstallOptions:     envtest.CRDInstallOptions{CleanUpAfterUse: true},
		CRDDirectoryPaths: crdPaths,
	}

	if _, err := env.Start(); err != nil {
		err = kerrors.NewAggregate([]error{err, env.Stop()})
		panic(err)
	}

	objs := []client.Object{}
	if len(uncachedObjs) > 0 {
		objs = append(objs, uncachedObjs...)
	}

	options := manager.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: objs,
			},
		},
	}

	mgr, err := ctrl.NewManager(env.Config, options)
	if err != nil {
		klog.Fatalf("Failed to start testenv manager: %v", err)
	}

	return &Environment{
		Manager: mgr,
		Client:  mgr.GetClient(),
		Config:  mgr.GetConfig(),
		env:     env,
	}
}

// Start starts the manager.
func (e *Environment) Start(ctx context.Context) error {
	err := errAlreadyStarted

	e.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(ctx)
		e.cancelManager = cancel
		err = e.Manager.Start(ctx)
	})

	return err
}

// Stop stops the test environment.
func (e *Environment) Stop() error {
	err := errAlreadyStopped

	e.stopOnce.Do(func() {
		e.cancelManager()
		err = e.env.Stop()
	})

	return err
}

// CreateKubeconfigSecret generates a new Kubeconfig secret from the envtest config.
func (e *Environment) CreateKubeconfigSecret(ctx context.Context, cluster *clusterv1.Cluster) error {
	return e.Create(ctx, kubeconfig.GenerateSecret(cluster, kubeconfig.FromEnvTestConfig(e.Config, cluster)))
}

// Cleanup deletes all the given objects.
func (e *Environment) Cleanup(ctx context.Context, objs ...client.Object) error {
	errs := []error{}

	for _, o := range objs {
		err := e.Client.Delete(ctx, o)
		if apierrors.IsNotFound(err) {
			continue
		}

		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}

// CleanupAndWait deletes all the given objects and waits for the cache to be updated accordingly.
//
// NOTE: Waiting for the cache to be updated helps in preventing test flakes due to the cache sync delays.
func (e *Environment) CleanupAndWait(ctx context.Context, objs ...client.Object) error {
	if err := e.Cleanup(ctx, objs...); err != nil {
		return err
	}

	// Makes sure the cache is updated with the deleted object
	errs := []error{}

	for _, o := range objs {
		// Ignoring namespaces because in testenv the namespace cleaner is not running.
		if o.GetObjectKind().GroupVersionKind().GroupKind() == corev1.SchemeGroupVersion.WithKind("Namespace").GroupKind() {
			continue
		}

		oCopy, ok := o.DeepCopyObject().(client.Object)
		if !ok {
			return fmt.Errorf("object type is not client.Object")
		}

		key := client.ObjectKeyFromObject(o)
		err := wait.ExponentialBackoff(
			cacheSyncBackoff,
			func() (done bool, err error) {
				if err := e.Get(ctx, key, oCopy); err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					}

					return false, err
				}

				return false, nil
			})
		if err != nil {
			errs = append(errs, fmt.Errorf("key %s, %s is not being deleted from the testenv client cache: %w", o.GetObjectKind().GroupVersionKind().String(), key, err))
		}
	}

	return kerrors.NewAggregate(errs)
}

// CreateAndWait creates the given object and waits for the cache to be updated accordingly.
//
// NOTE: Waiting for the cache to be updated helps in preventing test flakes due to the cache sync delays.
func (e *Environment) CreateAndWait(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if err := e.Client.Create(ctx, obj, opts...); err != nil {
		return err
	}

	// Makes sure the cache is updated with the new object
	objCopy, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("object type is not client.Object")
	}

	key := client.ObjectKeyFromObject(obj)
	if err := wait.ExponentialBackoff(
		cacheSyncBackoff,
		func() (done bool, err error) {
			if err := e.Get(ctx, key, objCopy); err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}

				return false, err
			}

			return true, nil
		}); err != nil {
		return fmt.Errorf("object %s, %s is not being added to the testenv client cache: %w", obj.GetObjectKind().GroupVersionKind().String(), key, err)
	}

	return nil
}

// CreateNamespace creates a new namespace with a generated name.
func (e *Environment) CreateNamespace(ctx context.Context, generateName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", generateName),
			Labels: map[string]string{
				"testenv/original-name": generateName,
			},
		},
	}
	if err := e.Client.Create(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func (e *Environment) EnsureNamespaceExists(ctx context.Context, namespace string) error {
	newNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if err := e.Client.Create(ctx, newNamespace); client.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("unable to create namespace %s: %w", namespace, err)
	}

	return nil
}

func getFilePathToClusterctlCRDs(root string) string {
	if clusterctlCRDPath := os.Getenv("CLUSTERCTL_CRD_PATH"); clusterctlCRDPath != "" {
		return clusterctlCRDPath
	}

	modBits, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	var clusterAPIVersion string

	for _, line := range strings.Split(string(modBits), "\n") {
		matches := clusterAPIVersionRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			clusterAPIVersion = matches[2]
		}
	}

	if clusterAPIVersion == "" {
		return ""
	}

	gopath := envOr("GOPATH", build.Default.GOPATH)

	return filepath.Join(gopath, "pkg", "mod", "sigs.k8s.io", fmt.Sprintf("cluster-api@v%s", clusterAPIVersion), "cmd", "clusterctl", "config", "crd", "bases")
}

func envOr(envKey, defaultValue string) string {
	if value, ok := os.LookupEnv(envKey); ok {
		return value
	}

	return defaultValue
}
