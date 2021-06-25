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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v1"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
)

// SkaffoldRunner is responsible for running the skaffold build, test and deploy config.
type SkaffoldRunner struct {
	runner.Builder
	runner.Pruner
	test.Tester

	deployer deploy.Deployer
	monitor  filemon.Monitor
	listener runner.Listener

	kubectlCLI         *kubectl.CLI
	cache              cache.Cache
	changeSet          runner.ChangeSet
	runCtx             *runcontext.RunContext
	labeller           *label.DefaultLabeller
	artifactStore      build.ArtifactStore
	sourceDependencies graph.SourceDependenciesCache
	// podSelector is used to determine relevant pods for logging and portForwarding
	podSelector kubernetes.ImageListMux

	devIteration int
	isLocalImage func(imageName string) (bool, error)
	hasDeployed  bool
	intents      *runner.Intents
	configs      []*latestV1.SkaffoldConfig
}

// HasDeployed returns true if this runner has deployed something.
func (r *SkaffoldRunner) HasDeployed() bool {
	return r.hasDeployed
}

func (r *SkaffoldRunner) SetV1Config(config []*latestV1.SkaffoldConfig) {
	r.configs = config
}

func (r *SkaffoldRunner) SetV2Config(config []*latestV2.SkaffoldConfig) {
	panic("skaffold runner v1 shall not use V2 config.")
}

func TargetArtifacts(configs []*latestV1.SkaffoldConfig, opts config.SkaffoldOptions) []*latestV1.Artifact {
	var targetArtifacts []*latestV1.Artifact
	for _, cfg := range configs {
		for _, artifact := range cfg.Build.Artifacts {
			if opts.IsTargetImage(artifact) {
				targetArtifacts = append(targetArtifacts, artifact)
			}
		}
	}
	return targetArtifacts
}

func (r *SkaffoldRunner) GetArtifacts() []*latestV1.Artifact {
	var artifacts []*latestV1.Artifact
	for _, cfg := range r.configs {
		artifacts = append(artifacts, cfg.Build.Artifacts...)
	}
	return artifacts
}

func (r *SkaffoldRunner) GetInsecureRegistries() []string {
	var regList []string
	for _, cfg := range r.configs {
		regList = append(regList, cfg.Build.InsecureRegistries...)
	}
	return regList
}
