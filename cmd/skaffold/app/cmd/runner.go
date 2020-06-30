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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
)

// For tests
var createRunner = createNewRunner

func withRunner(ctx context.Context, action func(runner.Runner, *latest.SkaffoldConfig) error) error {
	runner, config, err := createRunner(opts)
	sErrors.SetSkaffoldOptions(opts)
	if err != nil {
		return err
	}

	err = action(runner, config)

	return alwaysSucceedWhenCancelled(ctx, err)
}

// createNewRunner creates a Runner and returns the SkaffoldConfig associated with it.
func createNewRunner(opts config.SkaffoldOptions) (runner.Runner, *latest.SkaffoldConfig, error) {
	runCtx, config, err := runContext(opts)
	if err != nil {
		return nil, nil, err
	}

	runner, err := runner.NewForConfig(runCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("creating runner: %w", err)
	}

	return runner, config, nil
}

func runContext(opts config.SkaffoldOptions) (*runcontext.RunContext, *latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(opts.ConfigurationFile, latest.Version)
	if err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return nil, nil, fmt.Errorf("[%s] not found. You might need to run `skaffold init`", opts.ConfigurationFile)
		}

		// If the error is NOT that the file doesn't exist, then we warn the user
		// that maybe they are using an outdated version of Skaffold that's unable to read
		// the configuration.
		warnIfUpdateIsAvailable()
		return nil, nil, fmt.Errorf("parsing skaffold config: %w", err)
	}

	config := parsed.(*latest.SkaffoldConfig)

	if err = schema.ApplyProfiles(config, opts); err != nil {
		return nil, nil, fmt.Errorf("applying profiles: %w", err)
	}

	kubectx.ConfigureKubeConfig(opts.KubeConfig, opts.KubeContext, config.Deploy.KubeContext)

	if err := defaults.Set(config); err != nil {
		return nil, nil, fmt.Errorf("setting default values: %w", err)
	}

	if err := validation.Process(config); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	runCtx, err := runcontext.GetRunContext(opts, config.Pipeline)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	return runCtx, config, nil
}

func warnIfUpdateIsAvailable() {
	latest, current, versionErr := update.GetLatestAndCurrentVersion()
	if versionErr == nil && latest.GT(current) {
		logrus.Warnf("Your Skaffold version might be too old. Download the latest version (%s) from:\n  %s\n", latest, releaseURL(latest))
	}
}
