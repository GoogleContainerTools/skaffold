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

package test

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// NewNoopTester returns a Tester that does nothing.
func NewNoopTester(testCases *[]*latest.TestCase) (Tester, error) {
	return &noopTester{}, nil
}

type noopTester struct{}

func (t *noopTester) Test(context.Context, io.Writer, []build.Artifact) error {
	return nil
}

func (t *noopTester) TestDependencies() ([]string, error) {
	return nil, nil
}
