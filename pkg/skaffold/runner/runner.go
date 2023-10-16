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
	"errors"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/verify"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

// Runner is responsible for running the skaffold build, test and deploy config.
type Runner interface {
	Apply(context.Context, io.Writer) error
	ApplyDefaultRepo(tag string) (string, error)
	Build(context.Context, io.Writer, []*latest.Artifact) ([]graph.Artifact, error)
	Cleanup(context.Context, io.Writer, bool, manifest.ManifestListByConfig, string) error
	Dev(context.Context, io.Writer, []*latest.Artifact) error
	// Deploy and DeployAndLog: Do they need the `graph.Artifact` and could use render output.
	Deploy(context.Context, io.Writer, []graph.Artifact, manifest.ManifestListByConfig) error
	DeployAndLog(context.Context, io.Writer, []graph.Artifact, manifest.ManifestListByConfig) error
	GeneratePipeline(context.Context, io.Writer, []util.VersionedConfig, []string, string) error
	HasBuilt() bool
	DeployManifests() manifest.ManifestListByConfig
	Prune(context.Context, io.Writer) error

	Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error)
	Test(context.Context, io.Writer, []graph.Artifact) error
	Verify(context.Context, io.Writer, []graph.Artifact) error
	VerifyAndLog(context.Context, io.Writer, []graph.Artifact) error

	Exec(context.Context, io.Writer, []graph.Artifact, string) error
}

// SkaffoldRunner is responsible for running the skaffold build, test and deploy config.
type SkaffoldRunner struct {
	Builder
	Pruner
	tester test.Tester

	renderer      renderer.Renderer
	deployer      deploy.Deployer
	verifier      verify.Verifier
	actionsRunner ActionsRunner
	monitor       filemon.Monitor
	listener      Listener

	cache              cache.Cache
	changeSet          ChangeSet
	runCtx             *runcontext.RunContext
	labeller           *label.DefaultLabeller
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
	platforms          platform.Resolver

	devIteration    int
	isLocalImage    func(imageName string) (bool, error)
	deployManifests manifest.ManifestListByConfig
	intents         *Intents
}

// DeployManifests returns a list of manifest if this runner has deployed something.
func (r *SkaffoldRunner) DeployManifests() manifest.ManifestListByConfig {
	return r.deployManifests
}
