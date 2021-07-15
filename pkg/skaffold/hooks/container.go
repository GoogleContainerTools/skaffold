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

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// containerSelector represents a policy for selecting target containers for running a particular lifecycle hook
type containerSelector func(v1.Pod, v1.Container) (bool, error)

// imageSelector chooses containers that run the given image name
func imageSelector(image string) containerSelector {
	return func(p v1.Pod, c v1.Container) (bool, error) {
		if p.Status.Phase != v1.PodRunning {
			return false, nil
		}
		return c.Image == image, nil
	}
}

// containerHook represents a lifecycle hook to be executed inside a running container
type containerHook struct {
	cfg        latestV1.ContainerHook
	cli        *kubectl.CLI
	selector   containerSelector
	namespaces []string
}

// run executes the lifecycle hook inside the target container
func (h containerHook) run(ctx context.Context, out io.Writer) error {
	errs, ctx := errgroup.WithContext(ctx)

	client, err := kubernetesclient.Client()
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	for _, ns := range h.namespaces {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("getting pods for namespace %q: %w", ns, err)
		}

		for _, p := range pods.Items {
			for _, c := range p.Spec.Containers {
				if matched, err := h.selector(p, c); err != nil {
					return err
				} else if !matched {
					continue
				}
				args := []string{p.Name, "--namespace", p.Namespace, "-c", c.Name, "--"}
				args = append(args, h.cfg.Command...)
				cmd := h.cli.Command(ctx, "exec", args...)
				cmd.Stderr = out
				cmd.Stdout = out
				errs.Go(func() error {
					return util.RunCmd(cmd)
				})
			}
		}
	}

	if err := errs.Wait(); err != nil {
		return fmt.Errorf("container hook execution failed due to error %w", err)
	}
	return nil
}
