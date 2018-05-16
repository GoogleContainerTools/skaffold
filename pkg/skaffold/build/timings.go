/*
Copyright 2018 Google LLC

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

package build

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

// WithTimings creates a builder that logs the duration of the build.
func WithTimings(b Builder) Builder {
	return withTimings{
		Builder: b,
	}
}

type withTimings struct {
	Builder
}

func (w withTimings) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]Build, error) {
	start := time.Now()
	fmt.Fprintln(out, "Starting build...")

	bRes, err := w.Builder.Build(ctx, out, tagger, artifacts)
	if err != nil {
		return nil, err
	}

	fmt.Fprintln(out, "Build complete in", time.Since(start))

	return bRes, nil
}
