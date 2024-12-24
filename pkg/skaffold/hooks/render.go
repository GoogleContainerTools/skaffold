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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// for testing
var (
	NewRenderRunner = newRenderRunner
)

type RenderHookRunner interface {
	RunPreHooks(ctx context.Context, out io.Writer) error
	RunPostHooks(ctx context.Context, list manifest.ManifestList, out io.Writer) (manifest.ManifestList, error)
	GetConfigName() string
}

func newRenderRunner(r latest.RenderHooks, namespaces *[]string, opts RenderEnvOpts, configName string) RenderHookRunner {
	return renderRunner{r, configName, namespaces, opts, new(sync.Map)}
}

func NewRenderEnvOpts(kubeContext string, namespaces []string) RenderEnvOpts {
	return RenderEnvOpts{
		KubeContext: kubeContext,
		Namespaces:  strings.Join(namespaces, ","),
	}
}

type renderRunner struct {
	latest.RenderHooks
	configName        string
	namespaces        *[]string
	opts              RenderEnvOpts
	visitedContainers *sync.Map // maintain a list of previous iteration containers, so that they can be skipped
}

func (r renderRunner) GetConfigName() string {
	return r.configName
}

func (r renderRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreRender)
}

func (r renderRunner) RunPostHooks(ctx context.Context, list manifest.ManifestList, out io.Writer) (manifest.ManifestList, error) {
	logWriter := log.GetWriter()

	if len(r.PostHooks) > 0 {
		output.Default.Fprintln(logWriter, fmt.Sprintf("Starting %s hooks...", phases.PostRender))
	}
	updated, err := manifest.Load(list.Reader())
	if err != nil {
		return manifest.ManifestList{}, fmt.Errorf("failed to load manifest")
	}
	env := r.getEnv()
	for _, h := range r.PostHooks {
		if h.HostHook != nil {
			hook := hostHook{latest.HostHook{
				Command: h.HostHook.Command,
				OS:      h.HostHook.OS,
				Dir:     h.HostHook.Dir,
			}, env}
			var b bytes.Buffer
			if h.HostHook.WithChange {
				if err := hook.run(ctx, updated.Reader(), &b); err != nil {
					if errors.Is(err, &Skip{}) {
						continue
					}
					return manifest.ManifestList{}, err
				}
				if b.Len() == 0 {
					return manifest.ManifestList{}, fmt.Errorf("the length of stdout should be greater than 0 when using render post hook with change")
				}
				updated, err = manifest.Load(&b)
				if err != nil {
					return manifest.ManifestList{}, fmt.Errorf("failed to load manifest")
				}
			} else {
				if err := hook.run(ctx, updated.Reader(), logWriter); err != nil && !errors.Is(err, &Skip{}) {
					return manifest.ManifestList{}, err
				}
			}
		}
	}
	if len(r.PostHooks) > 0 {
		output.Default.Fprintln(logWriter, fmt.Sprintf("Completed %s hooks", phases.PostRender))
	}
	return updated, nil
}

func (r renderRunner) getEnv() []string {
	common := getEnv(staticEnvOpts)
	render := getEnv(r.opts)
	return append(common, render...)
}

func (r renderRunner) run(ctx context.Context, out io.Writer, hooks []latest.RenderHookItem, phase phase) error {
	logWriter := log.GetWriter()

	if len(hooks) > 0 {
		output.Default.Fprintln(logWriter, fmt.Sprintf("Starting %s hooks...", phase))
	}
	env := r.getEnv()
	for _, h := range hooks {
		if h.HostHook != nil {
			hook := hostHook{*h.HostHook, env}
			if err := hook.run(ctx, nil, out); err != nil && !errors.Is(err, &Skip{}) {
				return err
			}
		}
	}
	if len(hooks) > 0 {
		output.Default.Fprintln(logWriter, fmt.Sprintf("Completed %s hooks...", phase))
	}
	return nil
}
