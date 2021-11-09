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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/lint"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var outFormat string

func NewCmdLint() *cobra.Command {
	return NewCmd("lint").
		WithDescription("Lints skaffold app skaffold.yaml(s) to improve skaffold's - user-experience, performance, etc..").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &outFormat, Name: "format", DefValue: lint.PlainTextOutput,
				Usage: "Output format. One of: plain-text(default) or json"}}).
		Hidden().
		NoArgs(doLint)
}

func doLint(ctx context.Context, out io.Writer) error {
	// createRunner initializes state for objects (Docker client, etc.) lint uses
	_, _, runCtx, err := createRunner(ctx, out, opts)
	log.Entry(ctx).Debugf("starting skaffold lint with runCtx: %v", runCtx)
	if err != nil {
		return err
	}
	return lint.Lint(ctx, out, lint.Options{
		Filename:     opts.ConfigurationFile,
		RepoCacheDir: opts.RepoCacheDir,
		OutFormat:    outFormat,
		Modules:      opts.ConfigurationFilter,
		Profiles:     opts.Profiles,
	}, runCtx)
}
