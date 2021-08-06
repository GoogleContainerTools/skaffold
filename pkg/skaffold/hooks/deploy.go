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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewDeployRunner(cli *kubectl.CLI, d v1.DeployHooks, namespaces []string, formatter logger.Formatter, opts DeployEnvOpts) Runner {
	return deployRunner{d, cli, namespaces, formatter, opts, new(sync.Map)}
}

func NewDeployEnvOpts(runID string, kubeContext string, namespaces []string) DeployEnvOpts {
	return DeployEnvOpts{
		RunID:       runID,
		KubeContext: kubeContext,
		Namespaces:  strings.Join(namespaces, ","),
	}
}

type deployRunner struct {
	v1.DeployHooks
	cli         *kubectl.CLI
	namespaces  []string
	formatter   logger.Formatter
	opts        DeployEnvOpts
	visitedPods *sync.Map // maintain a list of previous iteration pods, so that they can be skipped
}

func (r deployRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreDeploy)
}

func (r deployRunner) RunPostHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PostHooks, phases.PostDeploy)
}

func (r deployRunner) getEnv() []string {
	common := getEnv(staticEnvOpts)
	deploy := getEnv(r.opts)
	return append(common, deploy...)
}

func (r deployRunner) run(ctx context.Context, out io.Writer, hooks []v1.DeployHookItem, phase phase) error {
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
		} else if h.ContainerHook != nil {
			hook := containerHook{
				cfg:        v1.ContainerHook{Command: h.ContainerHook.Command},
				cli:        r.cli,
				selector:   filterPodsSelector(r.visitedPods, phase, namePatternSelector(h.ContainerHook.PodName, h.ContainerHook.ContainerName)),
				namespaces: r.namespaces,
				formatter:  r.formatter,
			}
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

// filterPodsSelector filters the pods that have already been processed from a previous deploy iteration
func filterPodsSelector(visitedPods *sync.Map, phase phase, selector containerSelector) containerSelector {
	return func(p corev1.Pod, c corev1.Container) (bool, error) {
		key := fmt.Sprintf("%s:%s", phase, p.GetName())
		if _, found := visitedPods.LoadOrStore(key, struct{}{}); found {
			return false, nil
		}
		return selector(p, c)
	}
}
