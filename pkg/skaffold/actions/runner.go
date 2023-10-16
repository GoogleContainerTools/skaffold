/*
Copyright 2023 The Skaffold Authors

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

package actions

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/fatih/semgroup"
	"golang.org/x/sync/errgroup"

	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type Runner struct {
	// Map to access the associated Exec environment of a given action.
	execEnvByAction map[string]ExecEnv

	// List with all the created exec environments; this helps to execute all the actions in the same order always.
	orderedExecEnvs []ExecEnv

	// Map to access the list of associated actions of a given Execution environment.
	acsByExecEnv map[ExecEnv][]string
}

func NewRunner(execEnvByAction map[string]ExecEnv, orderedExecEnvs []ExecEnv, acsByExecEnv map[ExecEnv][]string) Runner {
	return Runner{execEnvByAction, orderedExecEnvs, acsByExecEnv}
}

func (r Runner) ExecAll(ctx context.Context, out io.Writer, allbuilds, localImgs []graph.Artifact) error {
	acs, err := r.prepareAllActions(ctx, out, allbuilds, localImgs)
	if err != nil {
		return err
	}

	defer r.cleanup(ctx, out, acs, r.orderedExecEnvs)
	return execWithFailingSafe(ctx, out, acs)
}

func (r Runner) Exec(ctx context.Context, out io.Writer, allbuilds, localImgs []graph.Artifact, aName string) error {
	execEnv, found := r.execEnvByAction[aName]
	if !found {
		return fmt.Errorf("custom action %v not found", aName)
	}

	output.Default.Fprintln(out, fmt.Sprintf("Starting execution for %v", aName))
	log.Entry(ctx).Debugf("Starting execution for %v", aName)

	acs, err := execEnv.PrepareActions(ctx, out, allbuilds, localImgs, []string{aName})
	if err != nil {
		return err
	}

	// We expect only one action to be created.
	if len(acs) != 1 {
		return fmt.Errorf("failed to create %v action", aName)
	}

	a := acs[0]

	err = a.Exec(ctx, out)
	log.Entry(ctx).Debugf("Finished execution for %v", a.name)
	r.cleanup(context.TODO(), out, []Task{a}, []ExecEnv{execEnv})
	return err
}

func (r Runner) prepareAllActions(ctx context.Context, out io.Writer, allbuilds, localImgs []graph.Artifact) ([]Task, error) {
	preparedAcs := []Task{}
	for _, execEnv := range r.orderedExecEnvs {
		acsNames := r.acsByExecEnv[execEnv]
		acs, err := execEnv.PrepareActions(ctx, out, allbuilds, localImgs, acsNames)
		if err != nil {
			return nil, err
		}

		for _, a := range acs {
			preparedAcs = append(preparedAcs, a)
		}
	}

	return preparedAcs, nil
}

func (r Runner) cleanup(ctx context.Context, out io.Writer, ts []Task, execEnvs []ExecEnv) {
	log.Entry(ctx).Debugf("Starting execution cleanup")

	for _, execEnv := range execEnvs {
		execEnv.Stop()
	}

	for _, t := range ts {
		if t == nil {
			continue
		}

		log.Entry(ctx).Debugf("Starting %v cleanup", t.Name())
		if err := t.Cleanup(ctx, out); err != nil {
			// TODO(renzor): known issue related with Docker client deleting containers + prune will cause some
			// warnings here. We need to fix https://github.com/GoogleContainerTools/skaffold/issues/8605
			log.Entry(ctx).Warnf("%v cleanup error:%v", t.Name(), err)
		}
		log.Entry(ctx).Debugf("Finished %v cleanup", t.Name())
	}
	for _, execEnv := range execEnvs {
		if execEnv == nil {
			continue
		}
		if err := execEnv.Cleanup(ctx, out); err != nil {
			log.Entry(ctx).Warnf("Execution environment cleanup error:%v", err)
		}
	}
}

func getExecFunc(isFailFast bool) ExecStrategy {
	if isFailFast {
		return execWithFailingFast
	}
	return execWithFailingSafe
}

func execWithFailingFast(ctx context.Context, out io.Writer, ts []Task) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return execAndLog(gCtx, out, t)
		})
	}

	return g.Wait()
}

func execWithFailingSafe(ctx context.Context, out io.Writer, ts []Task) error {
	const maxWorkers = math.MaxInt64
	g := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return execAndLog(ctx, out, t)
		})
	}

	return g.Wait()
}

func execAndLog(ctx context.Context, out io.Writer, t Task) error {
	log.Entry(ctx).Debugf("Starting execution for %v", t.Name())
	eventV2.CustomActionTaskInProgress(t.Name())
	err := t.Exec(ctx, out)

	if err != nil {
		eventV2.CustomActionTaskFailed(t.Name(), err)
	} else {
		eventV2.CustomActionTaskSucceeded(t.Name())
	}

	log.Entry(ctx).Debugf("Execution finished for %v", t.Name())
	return err
}
