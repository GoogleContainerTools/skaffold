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
package v2

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
)

type SkaffoldRunner struct {
	runner.Builder
	runner.Pruner
	test.Tester
	configs []*latestV2.SkaffoldConfig
}

func (r *SkaffoldRunner) HasDeployed() bool { return true }

func (r *SkaffoldRunner) SetV1Config(config []*latestV1.SkaffoldConfig) {
	panic("skaffold runner v2 shall not use V1 config.")
}

func (r *SkaffoldRunner) SetV2Config(config []*latestV2.SkaffoldConfig) {
	r.configs = config
}

func TargetArtifacts(configs []*latestV2.SkaffoldConfig, opts config.SkaffoldOptions) []*latestV1.Artifact {
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
