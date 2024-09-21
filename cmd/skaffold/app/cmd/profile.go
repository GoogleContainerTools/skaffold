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

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/cmd/profile"
)

func NewCmdProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Work with Skaffold profiles",
	}

	cmd.AddCommand(NewCmdProfileList())

	return cmd
}

func NewCmdProfileList() *cobra.Command {
	return NewCmd("list").
		WithDescription("List available profile names").
		WithFlagAdder(profile.AddListFlags).
		NoArgs(profile.List)
}
