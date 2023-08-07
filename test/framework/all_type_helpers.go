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
	"context"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GetterInterface interface {
	GetReader() client.Reader
	GetObject() client.Object
}

type Condition = func() bool

type ConditionalInterface interface {
	GetterInterface
	Satifies() bool
}

type ConditionalInput struct {
	client.Reader
	client.Object
	Condition
}

func For(object client.Object) *ConditionalInput {
	return &ConditionalInput{
		Object: object,
	}
}

func (in *ConditionalInput) In(reader client.Reader) *ConditionalInput {
	in.Reader = reader
	return in
}

func (in *ConditionalInput) ToSatisfy(condition Condition) *ConditionalInput {
	in.Condition = condition
	return in
}

func (in ConditionalInput) Satifies() bool {
	if in.Condition == nil {
		return true
	}

	By("Waiting for the object to satisfy condition...")
	return in.Condition()
}

func (in ConditionalInput) GetReader() client.Reader {
	return in.Reader
}

func (in ConditionalInput) GetObject() client.Object {
	return in.Object
}

// WaitForDelete will wait for object removal
func WaitForDelete(ctx context.Context, input GetterInterface, intervals ...interface{}) {
	By(fmt.Sprintf("Waiting for the %s object to be removed...", client.ObjectKeyFromObject(input.GetObject())))
	Eventually(func() bool {
		if err := input.GetReader().Get(ctx, client.ObjectKeyFromObject(input.GetObject()), input.GetObject()); err != nil {
			if apierrors.IsNotFound(err) {
				return true
			}
			klog.Infof("Failed to get an object: %+v", err)
		}
		return false
	}, intervals...).Should(BeTrue(), "Failed to wait until object deletion %s", klog.KObj(input.GetObject()))
}

// WaitFor will wait for condition match on existing object
func WaitFor(ctx context.Context, input ConditionalInterface, intervals ...interface{}) {
	Eventually(func() bool {
		By(fmt.Sprintf("Waiting for %s...", client.ObjectKeyFromObject(input.GetObject())))
		if err := input.GetReader().Get(ctx, client.ObjectKeyFromObject(input.GetObject()), input.GetObject()); err != nil {
			klog.Infof("Failed to get an object: %+v", err)
			return false
		}
		return input.Satifies()
	}, intervals...).Should(BeTrue(), "Failed to wait until object condition match %s", klog.KObj(input.GetObject()))
}

type HelmOutput int

const (
	Manifests HelmOutput = iota
	Hooks
	Full
)

type HelmChart struct {
	BinaryPath      string
	Path            string
	Name            string
	Kubeconfig      string
	DryRun          bool
	Wait            bool
	AdditionalFlags []string
	Output          HelmOutput
}

// InstallChart performs an install of the helm chart. Install returns the rendered manifest
// with some additional data that can't be parsed as yaml. This function processes the output and returns only the optional resources,
// marked as post install hooks.
func (h *HelmChart) InstallChart(values map[string]string) (string, error) {
	args := []string{"install", "--kubeconfig", h.Kubeconfig, h.Name, h.Path}
	if h.DryRun {
		args = append(args, "--dry-run")
	}
	if h.Wait {
		args = append(args, "--wait")
	}
	for key, value := range values {
		args = append(args, "--set")
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}
	if h.AdditionalFlags != nil {
		args = append(args, h.AdditionalFlags...)
	}

	fullCommand := append([]string{h.BinaryPath}, args...)
	klog.Infof("Executing: %s", fullCommand, " ")
	cmd := exec.Command(h.BinaryPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run helm install: %w, output: %s", err, string(out))
	}

	outString := string(out)
	switch h.Output {
	case Full:
		return outString, nil
	case Hooks:
		startIndex := strings.Index(outString, "HOOKS:")
		endIndex := strings.Index(outString, "MANIFEST:")

		if startIndex != -1 && endIndex != -1 {
			res := outString[startIndex+len("HOOKS:") : endIndex]
			res = strings.TrimPrefix(res, "\n")
			res = strings.TrimSuffix(res, "\n")
			return res, nil
		}
	case Manifests:
		startIndex := strings.Index(outString, "MANIFEST:")
		if startIndex != -1 {
			res := outString[startIndex+len("MANIFEST:"):]
			res = strings.TrimPrefix(res, "\n")
			res = strings.TrimSuffix(res, "\n")
			return res, nil
		}
	}

	return "", fmt.Errorf("failed to parse helm output")
}
