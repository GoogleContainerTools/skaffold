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
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func NewSyncRunner(cli *kubectl.CLI, imageName, imageRef string, namespaces []string, d v1.SyncHooks, opts SyncEnvOpts) Runner {
	return syncRunner{d, cli, imageName, imageRef, namespaces, opts}
}
func NewSyncEnvOpts(a *v1.Artifact, image string, addOrModifyFiles []string, deleteFiles []string, namespaces []string, kubeContext string) (SyncEnvOpts, error) {
	workDir, err := filepath.Abs(a.Workspace)
	if err != nil {
		return SyncEnvOpts{}, fmt.Errorf("determining build workspace directory for image %v: %w", a.ImageName, err)
	}
	return SyncEnvOpts{
		Image:                image,
		BuildContext:         workDir,
		FilesAddedOrModified: util.StringPtr(strings.Join(addOrModifyFiles, ";")),
		FilesDeleted:         util.StringPtr(strings.Join(deleteFiles, ";")),
		KubeContext:          kubeContext,
		Namespaces:           strings.Join(namespaces, ","),
	}, nil
}

type syncRunner struct {
	v1.SyncHooks
	cli        *kubectl.CLI
	imageName  string
	imageRef   string
	namespaces []string
	opts       SyncEnvOpts
}

func (r syncRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreSync)
}

func (r syncRunner) RunPostHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PostHooks, phases.PostSync)
}

func (r syncRunner) getEnv() []string {
	common := getEnv(staticEnvOpts)
	sync := getEnv(r.opts)
	return append(common, sync...)
}

func (r syncRunner) run(ctx context.Context, out io.Writer, hooks []v1.SyncHookItem, phase phase) error {
	if len(hooks) > 0 {
		output.Default.Fprintf(out, "Starting %s hooks for artifact %q...\n", phase, r.imageName)
	}
	env := r.getEnv()
	for _, h := range hooks {
		if h.HostHook != nil {
			hook := hostHook{*h.HostHook, env}
			if err := hook.run(ctx, out); err != nil {
				return fmt.Errorf("failed to execute host %s hook for artifact %q: %w", phase, r.imageName, err)
			}
		} else if h.ContainerHook != nil {
			hook := containerHook{
				cfg:        *h.ContainerHook,
				cli:        r.cli,
				selector:   imageSelector(r.imageRef),
				namespaces: r.namespaces,
			}
			if err := hook.run(ctx, out); err != nil {
				return fmt.Errorf("failed to execute container %s hook for artifact %q: %w", phase, r.imageName, err)
			}
		}
	}
	if len(hooks) > 0 {
		output.Default.Fprintf(out, "Completed %s hooks for artifact %q\n", phase, r.imageName)
	}
	return nil
}
