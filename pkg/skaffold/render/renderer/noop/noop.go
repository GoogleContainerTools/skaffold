/*
Copyright 2022 The Skaffold Authors

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

package noop

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

type Noop struct {}

func New(config *latestV2.RenderConfig, workingDir, hydrationDir string, labels map[string]string) (Noop, error) {
	//generator := generate.NewGenerator(workingDir, config.Generate, hydrationDir)
	return Noop{}, nil
}

func (r Noop) Render(_ context.Context, _ io.Writer, _ []graph.Artifact, _ bool, _ string) error {
	return nil
}

func (r Noop) ManifestDeps() ([]string, error) {
	return nil, nil
}
