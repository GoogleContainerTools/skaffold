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
	"fmt"
	"io"
	"os"

	shell "github.com/kballard/go-shellquote"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

func Run(out, stderr io.Writer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	catchStackdumpRequests()
	catchCtrlC(cancel)

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
	err := c.ExecuteContext(ctx)
	return extractInvalidUsageError(err)
}
