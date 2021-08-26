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

package v1

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

func (r *SkaffoldRunner) Test(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if len(r.runCtx.Pipelines.TestCases()) == 0 {
		return nil
	}

	eventV2.TaskInProgress(constants.Test, "Test")
	out, ctx = output.WithEventContext(ctx, out, constants.Test, constants.SubtaskIDNone)

	if err := r.tester.Test(ctx, out, artifacts); err != nil {
		eventV2.TaskFailed(constants.Test, err)
		return err
	}

	eventV2.TaskSucceeded(constants.Test)
	return nil
}
