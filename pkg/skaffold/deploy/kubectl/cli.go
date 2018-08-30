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
	"bufio"
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CLI holds parameters to run kubectl.
type CLI struct {
	Namespace   string
	KubeContext string
	Flags       v1alpha2.KubectlFlags

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
	logrus.Debugln(len(manifests), "manifests to deploy.", len(manifests), "are updated or new")
	c.previousApply = manifests
	if len(updated) == 0 {
		return nil, nil
	}

	buf := bytes.NewBuffer([]byte{})
	writer := bufio.NewWriter(buf)
	if err := c.Run(ctx, manifests.Reader(), writer, "apply", c.Flags.Apply, "-f", "-"); err != nil {
		if !strings.Contains(buf.String(), "field is immutable") {
			return nil, err
		}
		// If the output contains the string 'field is immutable', we want to delete the object and recreate it
		// See Issue #891 for more information
		if err := c.Delete(ctx, out, manifests); err != nil {
			return nil, errors.Wrap(err, "deleting manifest")
		}
		if err := c.Run(ctx, manifests.Reader(), out, "apply", c.Flags.Apply, "-f", "-"); err != nil {
			return nil, errors.Wrap(err, "kubectl apply after deletion")
		}
	} else {
		// Write output to out
		if _, err := out.Write(buf.Bytes()); err != nil {
			return nil, errors.Wrap(err, "writing to out")
		}
	}

	return updated, nil
}

// Run shells out kubectl CLI.
func (c *CLI) Run(ctx context.Context, in io.Reader, out io.Writer, command string, commandFlags []string, arg ...string) error {
	args := []string{"--context", c.KubeContext}
	if c.Namespace != "" {
		args = append(args, "--namespace", c.Namespace)
	}
	args = append(args, c.Flags.Global...)
	args = append(args, command)
	args = append(args, commandFlags...)
	args = append(args, arg...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}
