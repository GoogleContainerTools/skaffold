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
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/gcb"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// builderCtx encapsulates a given skaffold run context along with additional builder constructs.
type builderCtx struct {
	*runcontext.RunContext
	artifactStore           build.ArtifactStore
	sourceDependenciesCache graph.TransitiveSourceDependenciesCache
}

func (b *builderCtx) ArtifactStore() build.ArtifactStore {
	return b.artifactStore
}

func (b *builderCtx) SourceDependenciesResolver() graph.TransitiveSourceDependenciesCache {
	return b.sourceDependenciesCache
}

// getBuilder creates a builder from a given RunContext and build pipeline type.
func getBuilder(r *runcontext.RunContext, s build.ArtifactStore, d graph.TransitiveSourceDependenciesCache, p latest.Pipeline) (build.PipelineBuilder, error) {
	bCtx := &builderCtx{artifactStore: s, sourceDependenciesCache: d, RunContext: r}
	switch {
	case p.Build.LocalBuild != nil:
		logrus.Debugln("Using builder: local")
		builder, err := local.NewBuilder(bCtx, p.Build.LocalBuild)
		if err != nil {
			return nil, err
		}
		return builder, nil

	case p.Build.GoogleCloudBuild != nil:
		logrus.Debugln("Using builder: google cloud")
		builder := gcb.NewBuilder(bCtx, p.Build.GoogleCloudBuild)
		return builder, nil

	case p.Build.Cluster != nil:
		logrus.Debugln("Using builder: cluster")
		builder, err := cluster.NewBuilder(bCtx, p.Build.Cluster)
		if err != nil {
			return nil, err
		}
		return builder, err

	default:
		return nil, fmt.Errorf("unknown builder for config %+v", p.Build)
	}
}
