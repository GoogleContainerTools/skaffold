/*
Copyright 2018 The Skaffold Authors

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

package kubectl

import (
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// CLI holds parameters to run kubectl.
type CLI struct {
	Namespace   string
	KubeContext string
	Flags       v1alpha2.KubectlFlags
}

// Run shells out kubectl CLI.
func (c *CLI) Run(in io.Reader, out io.Writer, command string, commandFlags []string, arg ...string) error {
	args := []string{"--context", c.KubeContext}
	if c.Namespace != "" {
		args = append(args, "--namespace", c.Namespace)
	}
	args = append(args, c.Flags.Global...)
	args = append(args, command)
	args = append(args, commandFlags...)
	args = append(args, arg...)

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}
