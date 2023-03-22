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
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/fatih/semgroup"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	actions    map[string]Action
	orderedAcs []string
}

func NewRunner(acs map[string]Action, orderedAcs []string) Runner {
	return Runner{actions: acs, orderedAcs: orderedAcs}
}

func (r Runner) ExecAll(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	if err := r.prepareExecEnvs(ctx, out, allbuilds); err != nil {
		return err
	}

	acs := r.allActions()
	var ts []Task
	for _, a := range acs {
		execF := r.getExecFunc(a)
		a.SetTasksExecFunc(execF)
		ts = append(ts, a)
	}

	return r.execParallelFailingSafe(ctx, ts)
}

func (r Runner) prepareExecEnvs(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	execEnvs, actionsXEnv := r.execEnvs()

	for _, execEnv := range execEnvs {
		acs := actionsXEnv[execEnv]
		if err := execEnv.Prepare(ctx, out, allbuilds, acs); err != nil {
			return err
		}
	}

	return nil
}

func (r Runner) allActions() (acs []Action) {
	for _, aName := range r.orderedAcs {
		acs = append(acs, r.actions[aName])
	}
	return
}

func (r Runner) execEnvs() ([]ExecEnv, map[ExecEnv][]Action) {
	actionsXEnv := map[ExecEnv][]Action{}
	execEnvs := []ExecEnv{}

	for _, aName := range r.orderedAcs {
		a := r.actions[aName]
		execEnv := a.ExecEnv()
		envs, found := actionsXEnv[execEnv]
		actionsXEnv[execEnv] = append(envs, a)
		if !found {
			execEnvs = append(execEnvs, execEnv)
		}
	}

	return execEnvs, actionsXEnv
}

func (r Runner) Exec(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, name string) error {
	a, found := r.actions[name]
	if !found {
		return fmt.Errorf("custom action not found")
	}

	execEnv := a.ExecEnv()
	if err := execEnv.Prepare(ctx, out, allbuilds, []Action{a}); err != nil {
		return err
	}

	execFunc := r.getExecFunc(a)
	return execFunc(ctx, a.Tasks())
}

func (r Runner) getExecFunc(a Action) ExecStrategy {
	if a.IsFailFast() {
		return r.execParallelFailingFast
	}
	return r.execParallelFailingSafe
}

func (r Runner) execParallelFailingFast(ctx context.Context, ts []Task) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return r.execute(gCtx, t)
		})
	}

	return g.Wait()
}

func (r Runner) execParallelFailingSafe(ctx context.Context, ts []Task) error {
	const maxWorkers = math.MaxInt64
	g := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return r.execute(ctx, t)
		})
	}

	return g.Wait()
}

func (r Runner) execute(ctx context.Context, t Task) error {
	var err error

	execCh := make(chan error)
	go func() {
		execCh <- t.Exec(ctx)
		close(execCh)
	}()

	select {
	case err = <-execCh:
		log.Entry(ctx).Debugf("finishing task execution %v\n", t.Name())

	case <-time.After(t.Timeout()):
		msg := fmt.Sprintf("timing out action %v", t.Name())
		log.Entry(ctx).Debugln(msg)
		err = fmt.Errorf(msg)

	case <-ctx.Done():
		log.Entry(ctx).Debugf("interrupting execution for task %v...\n", t.Name())
		err = ctx.Err()
	}

	return err
}
