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
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

var versionFlag = flags.NewTemplateFlag("{{.Version}}\n", version.Info{})

func NewCmdVersion() *cobra.Command {
	return NewCmd("version").
		WithDescription("Print the version information").
		WithFlags(func(f *pflag.FlagSet) {
			f.VarP(versionFlag, "output", "o", versionFlag.Usage())
		}).
		NoArgs(doVersion)
}

func doVersion(_ context.Context, out io.Writer) error {
	return versionFlag.Template().Execute(out, version.Get())
}
