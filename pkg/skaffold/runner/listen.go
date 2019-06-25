/*
Copyright 2019 The Skaffold Authors

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

package runner

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Listen listens to a trigger, and when one is received, computes file changes and
// conditionally runs the dev loop.
func (r *SkaffoldRunner) Listen(ctx context.Context, out io.Writer, onChange func() error) error {
	ctxTrigger, cancelTrigger := context.WithCancel(ctx)
	defer cancelTrigger()
	trigger, err := trigger.StartTrigger(ctxTrigger, r.Trigger)
	if err != nil {
		return errors.Wrap(err, "unable to start trigger")
	}

	// exit if file monitor fails the first time
	err = r.Monitor.Run(ctx, out, r.Trigger.Debounce())
	if err != nil {
		return errors.Wrap(err, "failed to monitor files")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-trigger:
			if err := r.Monitor.Run(ctx, out, r.Trigger.Debounce()); err != nil {
				logrus.Warnf("error computing file changes: %s", err.Error())
				logrus.Warnf("skaffold may not run successfully!")
			}
			if err := onChange(); err != nil {
				logrus.Errorf("error running dev loop: %s", err.Error())
			}
		}
	}
}
