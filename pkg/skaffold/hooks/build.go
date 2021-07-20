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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// BuildRunner creates a new runner for pre-build and post-build lifecycle hooks
func BuildRunner(d v1.BuildHooks, opts BuildEnvOpts) Runner {
	return buildRunner{BuildHooks: d, opts: opts}
}

// NewBuildEnvOpts returns `BuildEnvOpts` required to create a `Runner` for build lifecycle hooks
func NewBuildEnvOpts(a *v1.Artifact, image string, pushImage bool) (BuildEnvOpts, error) {
	ref, err := docker.ParseReference(image)
	if err != nil {
		return BuildEnvOpts{}, fmt.Errorf("parsing image %v: %w", image, err)
	}

	w, err := filepath.Abs(a.Workspace)
	if err != nil {
		return BuildEnvOpts{}, fmt.Errorf("determining build workspace directory for image %v: %w", a.ImageName, err)
	}
	return BuildEnvOpts{
		Image:        image,
		PushImage:    pushImage,
		ImageRepo:    ref.Repo,
		ImageTag:     ref.Tag,
		BuildContext: w,
	}, nil
}

type buildRunner struct {
	v1.BuildHooks
	opts BuildEnvOpts
}

func (r buildRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreBuild)
}

func (r buildRunner) RunPostHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PostHooks, phases.PostBuild)
}

func (r buildRunner) getEnv() []string {
	common := getEnv(staticEnvOpts)
	build := getEnv(r.opts)
	return append(common, build...)
}

func (r buildRunner) run(ctx context.Context, out io.Writer, hooks []v1.HostHook, phase phase) error {
	if len(hooks) > 0 {
		output.Default.Fprintln(out, fmt.Sprintf("Starting %s hooks...", phase))
	}
	env := r.getEnv()
	for _, h := range hooks {
		hook := hostHook{h, env}
		if err := hook.run(ctx, out); err != nil {
			return err
		}
	}
	if len(hooks) > 0 {
		output.Default.Fprintln(out, fmt.Sprintf("Completed %s hooks", phase))
	}
	return nil
}
