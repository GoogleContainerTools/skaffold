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
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

type Action struct {
	name     string
	timeout  time.Duration
	tasks    []Task
	execFunc ExecStrategy
}

func NewAction(name string, timeoutSeconds int, isFailFast bool, tasks []Task) *Action {
	return &Action{
		name:     name,
		timeout:  time.Duration(timeoutSeconds * int(time.Second)),
		execFunc: getExecFunc(isFailFast),
		tasks:    tasks,
	}
}

func (a Action) Name() string {
	return a.name
}

func (a Action) Exec(ctx context.Context, out io.Writer) error {
	ctxTimeout := ctx
	cancel := func() {}

	if a.timeout.Seconds() > 0 {
		ctxTimeout, cancel = context.WithTimeout(ctx, a.timeout)
	}

	defer cancel()
	return a.execFunc(ctxTimeout, out, a.tasks)
}

func (a Action) Cleanup(ctx context.Context, out io.Writer) error {
	for _, t := range a.tasks {
		log.Entry(ctx).Debugf("Starting %v task cleanup", t.Name())
		if err := t.Cleanup(ctx, out); err != nil {
			return err
		}
		log.Entry(ctx).Debugf("Finished %v task cleanup", t.Name())
	}
	return nil
}
