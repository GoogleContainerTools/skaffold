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
	"context"
	"io"
	"os/exec"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CLI holds parameters to run kubectl.
type CLI struct {
	Namespace   string
	KubeContext string
	Flags       latest.KubectlFlags

	version       ClientVersion
	versionOnce   sync.Once
	previousApply ManifestList
}

// Delete runs `kubectl delete` on a list of manifests.
func (c *CLI) Delete(ctx context.Context, out io.Writer, manifests ManifestList) error {
	if err := c.Run(ctx, manifests.Reader(), out, "delete", c.Flags.Delete, "--ignore-not-found=true", "-f", "-"); err != nil {
		return errors.Wrap(err, "kubectl delete")
	}

	return nil
}

// Apply runs `kubectl apply` on a list of manifests.
func (c *CLI) Apply(ctx context.Context, out io.Writer, manifests ManifestList) (ManifestList, error) {
	// Only redeploy modified or new manifests
	// TODO(dgageot): should we delete a manifest that was deployed and is not anymore?
	updated := c.previousApply.Diff(manifests)
	logrus.Debugln(len(manifests), "manifests to deploy.", len(updated), "are updated or new")
	c.previousApply = manifests
	if len(updated) == 0 {
		return nil, nil
	}

	// Add --force flag to delete and redeploy image if changes can't be applied
	if err := c.Run(ctx, updated.Reader(), out, "apply", c.Flags.Apply, "--force", "-f", "-"); err != nil {
		return nil, errors.Wrap(err, "kubectl apply")
	}

	return updated, nil
}

// ReadManifests reads a list of manifests in yaml format.
func (c *CLI) ReadManifests(ctx context.Context, manifests []string) (ManifestList, error) {
	var list []string
	for _, manifest := range manifests {
		list = append(list, "-f", manifest)
	}

	args := c.args("create", []string{"--dry-run", "-oyaml"}, list...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	buf, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "kubectl create")
	}

	var manifestList ManifestList
	manifestList.Append(buf)
	logrus.Debugln("manifests", manifestList.String())

	return manifestList, nil
}

// Run shells out kubectl CLI.
func (c *CLI) Run(ctx context.Context, in io.Reader, out io.Writer, command string, commandFlags []string, arg ...string) error {
	args := c.args(command, commandFlags, arg...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}

func (c *CLI) args(command string, commandFlags []string, arg ...string) []string {
	args := []string{"--context", c.KubeContext}
	if c.Namespace != "" {
		args = append(args, "--namespace", c.Namespace)
	}
	args = append(args, c.Flags.Global...)
	args = append(args, command)
	args = append(args, commandFlags...)
	args = append(args, arg...)

	return args
}
