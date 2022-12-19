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

package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/v2/cmd/skaffold/app/cmd"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

func Run(out, stderr io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)
	defer cancel()

	catchStackdumpRequests()

	c := cmd.NewSkaffoldCommand(out, stderr)
	if cmdLine := os.Getenv("SKAFFOLD_CMDLINE"); cmdLine != "" && len(os.Args) == 1 {
		parsed, err := shell.Split(cmdLine)
		if err != nil {
			return fmt.Errorf("SKAFFOLD_CMDLINE is invalid: %w", err)
		}
		// XXX logged before logrus.SetLevel is called in NewSkaffoldCommand's PersistentPreRunE
		log.Entry(ctx).Debugf("Retrieving command line from SKAFFOLD_CMDLINE: %q", parsed)
		c.SetArgs(parsed)
	}
	c, err := c.ExecuteContextC(ctx)
	if err != nil {
		err = extractInvalidUsageError(err)
		if errors.Is(err, context.Canceled) {
			log.Entry(ctx).Debugln("ignore error since context is cancelled:", err)
		} else if !cmd.ShouldSuppressErrorReporting(c) {
			// As we allow some color setup using CLI flags for the main run, we can't run SetupColors()
			// for the entire skaffold run here. It's possible SetupColors() was never called, so call it again
			// before we print an error to get the right coloring.
			errOut := output.SetupColors(context.Background(), stderr, output.DefaultColorCode, false)
			output.Red.Fprintln(errOut, err)
		}
	}
	return err
}
