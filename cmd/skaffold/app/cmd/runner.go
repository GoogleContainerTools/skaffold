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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// newRunner creates a SkaffoldRunner and returns the SkaffoldConfig associated with it.
func newRunner(opts *config.SkaffoldOptions) (*runner.SkaffoldRunner, *latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfig(opts.ConfigurationFile, true)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing skaffold config")
	}

	if err := schema.CheckVersionIsLatest(parsed.GetVersion()); err != nil {
		return nil, nil, errors.Wrap(err, "invalid config")
	}

	config := parsed.(*latest.SkaffoldConfig)
	err = schema.ApplyProfiles(config, opts.Profiles)
	if err != nil {
		return nil, nil, errors.Wrap(err, "applying profiles")
	}

	runner, err := runner.NewForConfig(opts, config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating runner")
	}

	return runner, config, nil
}
