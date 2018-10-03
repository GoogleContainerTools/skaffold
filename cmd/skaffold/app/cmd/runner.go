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

package cmd

import (
	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// newRunner creates a SkaffoldRunner and returns the SkaffoldPipeline associated with it.
func newRunner(opts *config.SkaffoldOptions) (*runner.SkaffoldRunner, *latest.SkaffoldPipeline, error) {
	parsed, err := schema.ParseConfig(opts.ConfigurationFile, true)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing skaffold config")
	}

	if err := schema.CheckVersionIsLatest(parsed.GetVersion()); err != nil {
		return nil, nil, errors.Wrap(err, "invalid config")
	}

	config := parsed.(*latest.SkaffoldPipeline)
	err = schema.ApplyProfiles(config, opts.Profiles)
	if err != nil {
		return nil, nil, errors.Wrap(err, "applying profiles")
	}

	globalConfig, err := configutil.GetConfigForKubectx()
	if err != nil {
		return nil, nil, errors.Wrap(err, "retrieving global config")
	}
	if err = applyDefaultRepoSubstitution(config, globalConfig); err != nil {
		return nil, nil, errors.Wrap(err, "substituting default repos")
	}

	runner, err := runner.NewForConfig(opts, config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating runner")
	}

	return runner, config, nil
}

func applyDefaultRepoSubstitution(config *latest.SkaffoldConfig, globalConfig *configutil.ContextConfig) error {
	if globalConfig == nil {
		// noop
		return nil
	}
	for _, artifact := range config.Build.Artifacts {
		artifact.ImageName = util.SubstituteDefaultRepoIntoImage(globalConfig.DefaultRepo, artifact.ImageName)
	}
	for _, testCase := range config.Test {
		testCase.ImageName = util.SubstituteDefaultRepoIntoImage(globalConfig.DefaultRepo, testCase.ImageName)
	}
	return nil
}
