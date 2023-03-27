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

	"github.com/fatih/semgroup"
	"golang.org/x/sync/errgroup"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type Runner struct {
	execEnvByAction map[string]ExecEnv
	orderedExecEnvs []ExecEnv
	acsByExecEnv    map[ExecEnv][]string
}

func NewRunner(execEnvByAction map[string]ExecEnv, orderedExecEnvs []ExecEnv, acsByExecEnv map[ExecEnv][]string) Runner {
	return Runner{execEnvByAction, orderedExecEnvs, acsByExecEnv}
}

func (r Runner) ExecAll(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) error {
	acs, err := r.prepareActions(ctx, out, allbuilds)
	if err != nil {
		return err
	}

	defer r.cleanup(ctx, out, acs, r.orderedExecEnvs)
	return r.execWithFailingSafe(ctx, out, acs)
}

func (r Runner) Exec(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, aName string) error {
	execEnv, found := r.execEnvByAction[aName]
	if !found {
		return fmt.Errorf("custom action %v not found", aName)
	}

	acs, err := execEnv.PrepareActions(ctx, out, allbuilds, []string{aName})
	if err != nil {
		return err
	}

	// We expect only one action to be created.
	if len(acs) != 1 {
		return fmt.Errorf("failed to create %v action", aName)
	}

	a := acs[0]
	a.SetExecFunc(r.getExecFunc(a))

	defer r.cleanup(ctx, out, []Task{a}, []ExecEnv{execEnv})
	return a.Exec(ctx, out)
}

func (r Runner) prepareActions(ctx context.Context, out io.Writer, allbuilds []graph.Artifact) ([]Task, error) {
	preparedAcs := []Task{}
	for _, execEnv := range r.orderedExecEnvs {
		acsNames := r.acsByExecEnv[execEnv]
		acs, err := execEnv.PrepareActions(ctx, out, allbuilds, acsNames)
		if err != nil {
			return nil, err
		}

		for _, a := range acs {
			a.SetExecFunc(r.getExecFunc(a))
			preparedAcs = append(preparedAcs, a)
		}
	}

	return preparedAcs, nil
}

func (r Runner) getExecFunc(a Action) ExecStrategy {
	if a.IsFailFast() {
		return r.execWithFailingFast
	}
	return r.execWithFailingSafe
}

func (r Runner) execWithFailingFast(ctx context.Context, out io.Writer, ts []Task) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return r.execute(gCtx, t, out)
		})
	}

	return g.Wait()
}

func (r Runner) execWithFailingSafe(ctx context.Context, out io.Writer, ts []Task) error {
	const maxWorkers = math.MaxInt64
	g := semgroup.NewGroup(context.Background(), maxWorkers)

	for _, t := range ts {
		t := t
		g.Go(func() error {
			return r.execute(ctx, t, out)
		})
	}

	return g.Wait()
}

func (r Runner) execute(ctx context.Context, t Task, out io.Writer) error {
	var err error
	execCh := make(chan error)

	go func() {
		execCh <- t.Exec(ctx, out)
		close(execCh)
	}()

	select {
	case err = <-execCh:
		log.Entry(ctx).Debugf("Finishing execution for %v", t.Name())

	case <-time.After(t.Timeout()):
		msg := fmt.Sprintf("timing out %v", t.Name())
		log.Entry(ctx).Debugf("Finishihin execution:%v", msg)
		err = fmt.Errorf(msg)
	}

	return err
}

func (r Runner) cleanup(ctx context.Context, out io.Writer, ts []Task, execEnvs []ExecEnv) {
	for _, t := range ts {
		if t == nil {
			continue
		}
		if err := t.Cleanup(ctx, out); err != nil {
			log.Entry(ctx).Debugf("%v cleanup error:%v", t.Name(), err)
		}
	}
	for _, execEnv := range execEnvs {
		if execEnv == nil {
			continue
		}
		if err := execEnv.Cleanup(ctx, out); err != nil {
			log.Entry(ctx).Debugf("Execution environment cleanup error:%v", err)
		}
	}
}
