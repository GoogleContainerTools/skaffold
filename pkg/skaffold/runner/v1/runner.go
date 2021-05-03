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
package v1

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/trigger"
)

// SkaffoldRunner is responsible for running the skaffold build, test and deploy config.
type SkaffoldRunner struct {
	*runner.Builder
	runner.Pruner
	runner.Tester
	*runner.ImageLoader

	builds   *[]graph.Artifact
	Deployer deploy.Deployer
	Syncer   sync.Syncer
	Monitor  filemon.Monitor
	Listener runner.Listener

	kubectlCLI         *kubectl.CLI
	cache              cache.Cache
	changeSet          runner.ChangeSet
	RunCtx             *runcontext.RunContext
	labeller           *label.DefaultLabeller
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
	// podSelector is used to determine relevant pods for logging and portForwarding
	podSelector *kubernetes.ImageList

	devIteration int
	isLocalImage func(imageName string) (bool, error)
	hasDeployed  bool
	intents      *runner.Intents
}

// for testing
var (
	newStatusCheck = status.NewStatusChecker
)

// HasDeployed returns true if this runner has deployed something.
func (r *SkaffoldRunner) HasDeployed() bool {
	return r.hasDeployed
}

// TODO:Move v1.SkaffoldRunner specific attributes from NewForConfig to NewSkaffoldRunner function.
func NewSkaffoldRunner(builder build.Builder, buildRunner *runner.Builder,
	deployer deploy.Deployer,
	syncer sync.Syncer,
	monitor filemon.Monitor,
	listener runner.Listener,
	trigger trigger.Trigger,
	pruner runner.Pruner,
	tester test.Tester,
	labeller *label.DefaultLabeller,
	podSelectors *kubernetes.ImageList,
	intents *runner.Intents,
	intentChan chan bool,
	artifactCache cache.Cache,
	runCtx *runcontext.RunContext,
	store build.ArtifactStore,
	sourceDependencies graph.SourceDependenciesCache,
	isLocalImage func(imageName string) (bool, error)) *SkaffoldRunner {
	kubectlCLI := pkgkubectl.NewCLI(runCtx, "")

	return &SkaffoldRunner{
		Builder:     buildRunner,
		ImageLoader: runner.NewImageLoader(buildRunner.GetBuilds(), runCtx, kubectlCLI),
		Pruner:      pruner,
		Tester:      runner.Tester{tester},
		builds:      buildRunner.GetBuilds(),
		Deployer:    deployer,
		Syncer:      syncer,
		Monitor:     monitor,
		Listener: &runner.SkaffoldListener{
			Monitor:                 monitor,
			Trigger:                 trigger,
			IntentChan:              intentChan,
			SourceDependenciesCache: sourceDependencies,
		},
		artifactStore:      store,
		sourceDependencies: sourceDependencies,
		kubectlCLI:         kubectlCLI,
		labeller:           labeller,
		podSelector:        podSelectors,
		cache:              artifactCache,
		intents:            intents,
		isLocalImage:       isLocalImage,
	}
}
