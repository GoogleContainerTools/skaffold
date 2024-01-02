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

package hooks

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
)

// hostHook represents a lifecycle hook to be executed on the host machine
type hostHook struct {
	cfg latest.HostHook
	env []string // environment variables to set in the hook process
}

type Skip struct{}

func (e *Skip) Error() string {
	return "host hook execution skipped."
}

// run executes the lifecycle hook on the host machine
func (h hostHook) run(ctx context.Context, in io.Reader, out io.Writer) error {
	if len(h.cfg.OS) > 0 && !stringslice.Contains(h.cfg.OS, runtime.GOOS) {
		log.Entry(ctx).Infof("host hook execution skipped due to OS criteria %q not matched for commands:\n%q\n", strings.Join(h.cfg.OS, ","), strings.Join(h.cfg.Command, " "))
		return &Skip{}
	}
	cmd := h.retrieveCmd(ctx, in, out)

	log.Entry(ctx).Debugf("Running command: %s", cmd.Args)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting cmd: %w", err)
	}
	return misc.HandleGracefulTermination(ctx, cmd)
}

func (h hostHook) retrieveCmd(ctx context.Context, in io.Reader, out io.Writer) *exec.Cmd {
	cmd := exec.CommandContext(ctx, h.cfg.Command[0], h.cfg.Command[1:]...)
	if in != nil {
		cmd.Stdin = in
	}
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = append(cmd.Env, h.env...)
	cmd.Env = append(cmd.Env, util.OSEnviron()...)
	cmd.Dir = h.cfg.Dir

	return cmd
}
