// Copyright 2019 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
)

// NewCmdTag creates a new cobra.Command for the tag subcommand.
func NewCmdTag(options *[]crane.Option) *cobra.Command {
	return &cobra.Command{
		Use:   "tag IMG TAG",
		Short: "Efficiently tag a remote image",
		Long: `Tag remote image without downloading it.

This differs slightly from the "copy" command in a couple subtle ways:

1. You don't have to specify the entire repository for the tag you're adding. For example, these two commands are functionally equivalent:
` + "```" + `
crane cp registry.example.com/library/ubuntu:v0 registry.example.com/library/ubuntu:v1
crane tag registry.example.com/library/ubuntu:v0 v1
` + "```" + `

2. We can skip layer existence checks because we know the manifest already exists. This makes "tag" slightly faster than "copy".`,
		Example: `# Add a v1 tag to ubuntu
crane tag ubuntu v1`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			img, tag := args[0], args[1]
			return crane.Tag(img, tag, *options...)
		},
	}
}
