/*
Copyright 2021 The Skaffold Authors

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
)

var inspectFlags = struct {
	fileName  string
	outFormat string
	modules   []string
	buildEnv  string
}{
	fileName: "skaffold.yaml",
}

func NewCmdInspect() *cobra.Command {
	return NewCmd("inspect").
		WithDescription("Helper commands for Cloud Code IDEs to interact with and modify skaffold configuration files.").
		WithPersistentFlagAdder(cmdInspectFlags).
		Hidden().
		WithCommands(cmdModules(), cmdProfiles())
}

func cmdInspectFlags(f *pflag.FlagSet) {
	f.StringVarP(&inspectFlags.fileName, "filename", "f", "skaffold.yaml", "Path to the local Skaffold config file. Defaults to `skaffold.yaml`")
	f.StringVarP(&inspectFlags.outFormat, "format", "o", "json", "Output format. One of: json(default)")
}
