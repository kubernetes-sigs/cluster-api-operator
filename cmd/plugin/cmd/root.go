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
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	configclient "sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	logf "sigs.k8s.io/cluster-api/cmd/clusterctl/log"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

var (
	cfgFile   string
	verbosity *int
)

// RootCmd is operator root CLI command.
var RootCmd = &cobra.Command{
	Use:   "operator",
	Short: "clusterctl plugin for leveraging Cluster API Operator.",
	Long: LongDesc(`
		Use this clusterctl plugin to bootstrap a management cluster for Cluster API with the Cluster API Operator.`),
}

// Execute executes the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		if verbosity != nil && *verbosity >= 5 {
			if err, ok := err.(stackTracer); ok { //nolint:errorlint
				for _, f := range err.StackTrace() {
					fmt.Fprintf(os.Stderr, "%+s:%d\n", f, f)
				}
			}
		}

		os.Exit(1)
	}
}

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	verbosity = flag.CommandLine.Int("v", 0, "Set the log level verbosity. This overrides the CLUSTERCTL_LOG_LEVEL environment variable.")

	RootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"Path to clusterctl configuration (default is `$XDG_CONFIG_HOME/cluster-api/clusterctl.yaml`) or to a remote location (i.e. https://example.com/clusterctl.yaml)")

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// check if the CLUSTERCTL_LOG_LEVEL was set via env var or in the config file
	if *verbosity == 0 { //nolint:nestif
		configClient, err := configclient.New(cfgFile)
		if err == nil {
			v, err := configClient.Variables().Get("CLUSTERCTL_LOG_LEVEL")
			if err == nil && v != "" {
				verbosityFromEnv, err := strconv.Atoi(v)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to convert CLUSTERCTL_LOG_LEVEL string to an int. err=%s\n", err.Error())
					os.Exit(1)
				}

				verbosity = &verbosityFromEnv
			}
		}
	}

	logf.SetLogger(logf.NewLogger(logf.WithThreshold(verbosity)))
}

const indentation = `  `

// LongDesc normalizes a command's long description to follow the conventions.
func LongDesc(s string) string {
	if s == "" {
		return s
	}

	return normalizer{s}.heredoc().trim().string
}

// Examples normalizes a command's examples to follow the conventions.
func Examples(s string) string {
	if s == "" {
		return s
	}

	return normalizer{s}.trim().indent().string
}

type normalizer struct {
	string
}

func (s normalizer) heredoc() normalizer {
	s.string = heredoc.Doc(s.string)
	return s
}

func (s normalizer) trim() normalizer {
	s.string = strings.TrimSpace(s.string)
	return s
}

func (s normalizer) indent() normalizer {
	splitLines := strings.Split(s.string, "\n")
	indentedLines := make([]string, 0, len(splitLines))

	for _, line := range splitLines {
		trimmed := strings.TrimSpace(line)
		indented := indentation + trimmed
		indentedLines = append(indentedLines, indented)
	}

	s.string = strings.Join(indentedLines, "\n")

	return s
}
