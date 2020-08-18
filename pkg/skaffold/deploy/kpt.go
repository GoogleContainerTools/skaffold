/*
Copyright 2020 The Skaffold Authors

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

package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// KptDeployer deploys workflows with kpt CLI
type KptDeployer struct {
	*latest.KptDeploy
}

func NewKptDeployer(runCtx *runcontext.RunContext, labels map[string]string) *KptDeployer {
	return &KptDeployer{
		KptDeploy: runCtx.Pipeline().Deploy.KptDeploy,
	}
}

func (k *KptDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	return nil, nil
}

func (k *KptDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (k *KptDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func (k *KptDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	return nil
}

// getApplyDir returns the path to applyDir if specified by the user. Otherwise, getApplyDir
// creates a hidden directory in place of applyDir.
func (k *KptDeployer) getApplyDir(ctx context.Context) (string, error) {
	if k.ApplyDir != "" {
		if _, err := os.Stat(k.ApplyDir); os.IsNotExist(err) {
			return "", err
		}
		return k.ApplyDir, nil
	}

	applyDir := ".kpt-hydrated"

	// 0755 is a permission setting where the owner can read, write, and execute.
	// Others can read and execute but not modify the file.
	if err := os.MkdirAll(applyDir, 0755); err != nil {
		return "", fmt.Errorf("applyDir was unspecified. creating applyDir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(applyDir, []string{"live", "init"}, nil, nil)...)
	util.RunCmd(cmd)

	return applyDir, nil
}

// kptCommandArgs returns a list of additional arguments for the kpt command.
func kptCommandArgs(dir string, commands, flags, globalFlags []string) []string {
	var args []string

	for _, v := range commands {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	if len(dir) > 0 {
		args = append(args, dir)
	}

	for _, v := range flags {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	for _, v := range globalFlags {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	return args
}
