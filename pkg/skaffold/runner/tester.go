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
package runner

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
)

type Tester struct {
	tester test.Tester
}

// Test tests a list of already built artifacts.
func (r *Tester) Test(ctx context.Context, out io.Writer, artifacts []graph.Artifact) error {
	if err := r.tester.Test(ctx, out, artifacts); err != nil {
		return err
	}

	return nil
}
