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
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
)

var versionDefaultTemplate = "{{.Version}}\n"
var versionFlag = flags.NewTemplateFlag(versionDefaultTemplate, version.Info{})

func NewCmdVersion() *cobra.Command {
	return NewCmd("version").
		WithDescription("Print the version information").
		WithFlags([]*Flag{
			{Value: versionFlag, Name: "output", Shorthand: "o", DefValue: versionDefaultTemplate, Usage: versionFlag.Usage()},
		}).
		NoArgs(doVersion)
}

func doVersion(_ context.Context, out io.Writer) error {
	return versionFlag.Template().Execute(out, version.Get())
}
