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
	"errors"
	"fmt"
	"io"
	"sync"

	corev1 "k8s.io/api/core/v1"

	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// for testing
var (
	NewDeployRunner         = newDeployRunner
	NewCloudRunDeployRunner = newCloudRunDeployRunner
)

func newDeployRunner(cli *kubectl.CLI, d latest.DeployHooks, namespaces *[]string, formatter logger.Formatter, opts DeployEnvOpts, manifestsNamespaces *[]string) Runner {
	return deployRunner{d, cli, namespaces, manifestsNamespaces, formatter, opts, new(sync.Map)}
}

func newCloudRunDeployRunner(d latest.CloudRunDeployHooks, opts DeployEnvOpts) Runner {
	deployHooks := latest.DeployHooks{}
	deployHooks.PreHooks = createDeployHostHooksFromCloudRunHooks(d.PreHooks)
	deployHooks.PostHooks = createDeployHostHooksFromCloudRunHooks(d.PostHooks)

	return deployRunner{
		DeployHooks: deployHooks,
		opts:        opts,
	}
}

func createDeployHostHooksFromCloudRunHooks(cloudRunHook []latest.HostHook) []latest.DeployHookItem {
	deployHooks := []latest.DeployHookItem{}

	for i := range cloudRunHook {
		hookItem := latest.DeployHookItem{
			HostHook: &cloudRunHook[i],
		}
		deployHooks = append(deployHooks, hookItem)
	}

	return deployHooks
}

func NewDeployEnvOpts(runID string, kubeContext string, namespaces []string) DeployEnvOpts {
	return DeployEnvOpts{
		RunID:       runID,
		KubeContext: kubeContext,
		Namespaces:  namespaces,
	}
}

type deployRunner struct {
	latest.DeployHooks
	cli                 *kubectl.CLI
	namespaces          *[]string
	manifestsNamespaces *[]string
	formatter           logger.Formatter
	opts                DeployEnvOpts
	visitedContainers   *sync.Map // maintain a list of previous iteration containers, so that they can be skipped
}

func (r deployRunner) RunPreHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PreHooks, phases.PreDeploy, nil)
}

func (r deployRunner) RunPostHooks(ctx context.Context, out io.Writer) error {
	return r.run(ctx, out, r.PostHooks, phases.PostDeploy, r.manifestsNamespaces)
}

func (r deployRunner) getEnv(manifestsNs *[]string) []string {
	mergedOpts := r.opts

	if manifestsNs != nil {
		mergedOpts.Namespaces = deployutil.ConsolidateNamespaces(r.opts.Namespaces, *manifestsNs)
	}

	common := getEnv(staticEnvOpts)
	deploy := getEnv(mergedOpts)
	return append(common, deploy...)
}

func (r deployRunner) run(ctx context.Context, out io.Writer, hooks []latest.DeployHookItem, phase phase, manifestsNs *[]string) error {
	if len(hooks) > 0 {
		output.Default.Fprintln(out, fmt.Sprintf("Starting %s hooks...", phase))
	}
	env := r.getEnv(manifestsNs)
	for _, h := range hooks {
		if h.HostHook != nil {
			hook := hostHook{*h.HostHook, env}
			if err := hook.run(ctx, nil, out); err != nil && !errors.Is(err, &Skip{}) {
				return err
			}
		} else if h.ContainerHook != nil {
			hook := containerHook{
				cfg:        latest.ContainerHook{Command: h.ContainerHook.Command},
				cli:        r.cli,
				selector:   filterContainersSelector(r.visitedContainers, phase, namePatternSelector(h.ContainerHook.PodName, h.ContainerHook.ContainerName)),
				namespaces: *r.namespaces,
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

// filterContainersSelector filters the containers that have already been processed from a previous deploy iteration
func filterContainersSelector(visitedContainers *sync.Map, phase phase, selector containerSelector) containerSelector {
	return func(p corev1.Pod, c corev1.Container) (bool, error) {
		key := fmt.Sprintf("%s:%s:%s", phase, p.GetName(), c.Name)
		if _, found := visitedContainers.LoadOrStore(key, struct{}{}); found {
			return false, nil
		}
		return selector(p, c)
	}
}
