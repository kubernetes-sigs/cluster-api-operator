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
	"context"

	"github.com/go-errors/errors"
	"github.com/spf13/cobra"
)

type moveOptions struct {
	fromKubeconfig        string
	fromKubeconfigContext string
	toKubeconfig          string
	toKubeconfigContext   string
	namespace             string
	fromDirectory         string
	toDirectory           string
	dryRun                bool
}

var moveOpts = &moveOptions{}

var moveCmd = &cobra.Command{
	Use:     "move",
	GroupID: groupManagement,
	Short:   "Move Cluster API objects and all dependencies between management clusters",
	Long: LongDesc(`
		Move Cluster API objects and all dependencies between management clusters.

		Note: The destination cluster MUST have the required provider components installed.`),

	Example: Examples(`
		Move Cluster API objects and all dependencies between management clusters.
		capioperator move --to-kubeconfig=target-kubeconfig.yaml

		Write Cluster API objects and all dependencies from a management cluster to directory.
		capioperator move --to-directory /tmp/backup-directory

		Read Cluster API objects and all dependencies from a directory into a management cluster.
		capioperator move --from-directory /tmp/backup-directory
	`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMove()
	},
}

func init() {
	moveCmd.Flags().StringVar(&moveOpts.fromKubeconfig, "kubeconfig", "",
		"Path to the kubeconfig file for the source management cluster. If unspecified, default discovery rules apply.")
	moveCmd.Flags().StringVar(&moveOpts.toKubeconfig, "to-kubeconfig", "",
		"Path to the kubeconfig file to use for the destination management cluster.")
	moveCmd.Flags().StringVar(&moveOpts.fromKubeconfigContext, "kubeconfig-context", "",
		"Context to be used within the kubeconfig file for the source management cluster. If empty, current context will be used.")
	moveCmd.Flags().StringVar(&moveOpts.toKubeconfigContext, "to-kubeconfig-context", "",
		"Context to be used within the kubeconfig file for the destination management cluster. If empty, current context will be used.")
	moveCmd.Flags().StringVarP(&moveOpts.namespace, "namespace", "n", "",
		"The namespace where the workload cluster is hosted. If unspecified, the current context's namespace is used.")
	moveCmd.Flags().BoolVar(&moveOpts.dryRun, "dry-run", false,
		"Enable dry run, don't really perform the move actions")
	moveCmd.Flags().StringVar(&moveOpts.toDirectory, "to-directory", "",
		"Write Cluster API objects and all dependencies from a management cluster to directory.")
	moveCmd.Flags().StringVar(&moveOpts.fromDirectory, "from-directory", "",
		"Read Cluster API objects and all dependencies from a directory into a management cluster.")

	moveCmd.MarkFlagsMutuallyExclusive("to-directory", "to-kubeconfig")
	moveCmd.MarkFlagsMutuallyExclusive("from-directory", "to-directory")
	moveCmd.MarkFlagsMutuallyExclusive("from-directory", "kubeconfig")

	RootCmd.AddCommand(moveCmd)
}

func runMove() error {
	ctx := context.Background()

	if moveOpts.toDirectory == "" &&
		moveOpts.fromDirectory == "" &&
		moveOpts.toKubeconfig == "" &&
		!moveOpts.dryRun {
		return errors.New("please specify a target cluster using the --to-kubeconfig flag when not using --dry-run, --to-directory or --from-directory")
	}

	return moveProvider(ctx, moveOpts)
}

func moveProvider(ctx context.Context, opts *moveOptions) error {
	return errors.New("Not implemented")
}
