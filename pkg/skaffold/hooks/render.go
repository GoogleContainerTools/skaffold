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
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// for testing
var (
	NewRenderRunner = newRenderRunner
)

func newRenderRunner(r latest.RenderHooks, namespaces *[]string, opts RenderEnvOpts) Runner {
	return renderRunner{r, namespaces, opts, new(sync.Map)}
}

func NewRenderEnvOpts(kubeContext string, namespaces []string) RenderEnvOpts {
	return RenderEnvOpts{
		KubeContext: kubeContext,
		Namespaces:  strings.Join(namespaces, ","),
	}
}

type renderRunner struct {
	latest.RenderHooks
	namespaces        *[]string
	opts              RenderEnvOpts
	visitedContainers *sync.Map // maintain a list of previous iteration containers, so that they can be skipped
}

func (r renderRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreRender)
}

func (r renderRunner) RunPostHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PostHooks, phases.PostRender)
}

func (r renderRunner) getEnv() []string {
	common := getEnv(staticEnvOpts)
	render := getEnv(r.opts)
	return append(common, render...)
}

func (r renderRunner) run(ctx context.Context, out io.Writer, hooks []latest.RenderHookItem, phase phase) error {
	if len(hooks) > 0 {
		output.Default.Fprintln(out, fmt.Sprintf("Starting %s hooks...", phase))
	}
	env := r.getEnv()
	for _, h := range hooks {
		if h.HostHook != nil {
			hook := hostHook{*h.HostHook, env}
			if err := hook.run(ctx, out); err != nil {
				return err
			}
		}
	}
	if len(hooks) > 0 {
		output.Default.Fprintln(out, fmt.Sprintf("Completed %s hooks", phase))
	}
	return nil
}
