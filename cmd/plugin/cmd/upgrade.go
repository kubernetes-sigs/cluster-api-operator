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
	"sort"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	GroupID: groupManagement,
	Short:   "Upgrade core and provider components in a management cluster",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	upgradeCmd.AddCommand(upgradePlanCmd)
	upgradeCmd.AddCommand(upgradeApplyCmd)
	RootCmd.AddCommand(upgradeCmd)
}

func sortUpgradeItems(plan upgradePlan) {
	sort.Slice(plan.Providers, func(i, j int) bool {
		return plan.Providers[i].Type < plan.Providers[j].Type ||
			(plan.Providers[i].Type == plan.Providers[j].Type && plan.Providers[i].Name < plan.Providers[j].Name) ||
			(plan.Providers[i].Type == plan.Providers[j].Type && plan.Providers[i].Name == plan.Providers[j].Name && plan.Providers[i].Namespace < plan.Providers[j].Namespace)
	})
}

func prettifyTargetVersion(version string) string {
	if version == "" {
		return "Already up to date"
	}

	return version
}
