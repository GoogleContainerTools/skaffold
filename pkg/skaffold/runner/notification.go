/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
)

const (
	// terminalBell is the sequence that triggers a beep in the terminal
	terminalBell = "\007"
)

// WithNotification creates a deployer that bips each time a deploy is done.
func WithNotification(d deploy.Deployer) deploy.Deployer {
	return withNotification{
		Deployer: d,
	}
}

type withNotification struct {
	deploy.Deployer
}

func (w withNotification) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []deploy.Labeller) error {
	if err := w.Deployer.Deploy(ctx, out, builds, labellers); err != nil {
		return err
	}

	fmt.Fprint(out, terminalBell)

	return nil
}
