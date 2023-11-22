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
	"errors"
	"flag"
	"os"
	"strings"

	logf "sigs.k8s.io/cluster-api/cmd/clusterctl/log"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/MakeNowJust/heredoc"
	goerrors "github.com/go-errors/errors"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

const (
	groupDebug      = "group-debug"
	groupManagement = "group-management"
	groupOther      = "group-other"
	latestVersion   = "latest"
)

var verbosity *int

var log logr.Logger

// RootCmd is capioperator root CLI command.
var RootCmd = &cobra.Command{
	Use:          "capioperator",
	SilenceUsage: true,
	Short:        "capioperator controls the lifecycle of a Cluster API management cluster",
	Long: LongDesc(`
		Get started with Cluster API using capioperator to create a management cluster,
		install providers, and create templates for your workload cluster.`),
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// Execute executes the root command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		if verbosity != nil && *verbosity >= 5 {
			var stackErr *goerrors.Error
			if errors.As(err, &stackErr) {
				stackErr.ErrorStack()
			}
		}
		// TODO: print cmd help if validation error
		os.Exit(1)
	}
}

func init() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	verbosity = flag.CommandLine.Int("v", 0, "Set the log level verbosity. This overrides the CAPIOPERATOR_LOG_LEVEL environment variable.")

	log = logf.NewLogger(logf.WithThreshold(verbosity))
	logf.SetLogger(log)
	ctrl.SetLogger(log)

	RootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	RootCmd.AddGroup(
		&cobra.Group{
			ID:    groupManagement,
			Title: "Cluster Management Commands:",
		},
		&cobra.Group{
			ID:    groupDebug,
			Title: "Troubleshooting and Debugging Commands:",
		},
		&cobra.Group{
			ID:    groupOther,
			Title: "Other Commands:",
		})

	RootCmd.SetHelpCommandGroupID(groupOther)
	RootCmd.SetCompletionCommandGroupID(groupOther)
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
