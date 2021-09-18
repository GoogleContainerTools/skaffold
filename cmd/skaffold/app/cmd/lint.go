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
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/lint"
)

var lintFlags = struct {
	filename     string
	outFormat    string
	modules      []string
	repoCacheDir string
	buildEnv     string
	profiles     []string
	strict       bool
}{
	filename: "skaffold.yaml",
	strict:   true,
}

func NewCmdLint() *cobra.Command {
	return NewCmd("lint").
		WithDescription("Lints skaffold app configuration files (eg: skaffold.yaml(s), Dockerfile(s), k8s manifests, etc.) to improve skaffold's - user-experience, performance, etc..").
		WithFlagAdder(cmdLintFlags).
		Hidden().
		NoArgs(rec)
}

func cmdLintFlags(f *pflag.FlagSet) {
	f.StringVarP(&lintFlags.filename, "filename", "f", "skaffold.yaml", "Path to the local Skaffold config file. Defaults to `skaffold.yaml`")
	f.StringVarP(&lintFlags.outFormat, "format", "o", "plain-text", "Output format. One of: plain-text(default) or json")
	f.StringVar(&lintFlags.repoCacheDir, "remote-cache-dir", "", "Specify the location of the remote git repositories cache (defaults to $HOME/.skaffold/repos)")
	f.StringSliceVarP(&lintFlags.profiles, "profile", "p", nil, `Profile names to activate`)
	f.StringSliceVarP(&lintFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func rec(ctx context.Context, out io.Writer) error {
	return lint.Lint(ctx, out, inspect.Options{
		Filename:     lintFlags.filename,
		RepoCacheDir: lintFlags.repoCacheDir,
		OutFormat:    lintFlags.outFormat,
		Modules:      lintFlags.modules,
		LintOptions:  inspect.LintOptions{LintProfiles: inspectFlags.profiles},
	})
}
