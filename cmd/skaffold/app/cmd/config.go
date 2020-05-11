/*
Copyright 2019 The Skaffold Authors

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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interact with the Skaffold configuration",
	}

	cmd.AddCommand(NewCmdSet())
	cmd.AddCommand(NewCmdUnset())
	cmd.AddCommand(NewCmdList())
	return cmd
}

func NewCmdSet() *cobra.Command {
	return NewCmd("set").
		WithDescription("Set a value in the global Skaffold config").
		WithExample("Mark a registry as insecure", "config set insecure-registries <insecure1.io>").
		WithExample("Globally set the default image repository", "config set default-repo <myrepo>").
		WithExample("Disable pushing images for a given Kubernetes context", "config set --kube-context <mycluster> local-cluster true").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddSetUnsetFlags(f)
		}).
		ExactArgs(2, config.Set)
}

func NewCmdUnset() *cobra.Command {
	return NewCmd("unset").
		WithDescription("Unset a value in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddSetUnsetFlags(f)
		}).
		ExactArgs(1, config.Unset)
}

func NewCmdList() *cobra.Command {
	return NewCmd("list").
		WithDescription("List all values set in the global Skaffold config").
		WithFlags(func(f *pflag.FlagSet) {
			config.AddCommonFlags(f)
			config.AddListFlags(f)
		}).
		NoArgs(config.List)
}
