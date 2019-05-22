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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// BuildAndTest builds artifacts and runs tests on built artifacts
func (r *SkaffoldRunner) BuildAndTest(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	tags, err := r.imageTags(ctx, out, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "generating tag")
	}
	r.hasBuilt = true

	artifactsToBuild, res, err := r.cache.RetrieveCachedArtifacts(ctx, out, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving cached artifacts")
	}

	bRes, err := r.Build(ctx, out, tags, artifactsToBuild)
	if err != nil {
		return nil, errors.Wrap(err, "build failed")
	}
	r.cache.RetagLocalImages(ctx, out, artifactsToBuild, bRes)
	bRes = append(bRes, res...)
	if err := r.cache.CacheArtifacts(ctx, artifacts, bRes); err != nil {
		logrus.Warnf("error caching artifacts: %v", err)
	}
	if !r.runCtx.Opts.SkipTests {
		if err = r.Test(ctx, out, bRes); err != nil {
			return nil, errors.Wrap(err, "test failed")
		}
	}
	return bRes, err
}
