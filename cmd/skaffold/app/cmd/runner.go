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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
)

// For tests
var createRunner = createNewRunner

func withRunner(ctx context.Context, action func(runner.Runner, []*latest.SkaffoldConfig) error) error {
	runner, config, err := createRunner(opts)
	sErrors.SetSkaffoldOptions(opts)
	if err != nil {
		return err
	}

	err = action(runner, config)

	return alwaysSucceedWhenCancelled(ctx, err)
}

// createNewRunner creates a Runner and returns the SkaffoldConfig associated with it.
func createNewRunner(opts config.SkaffoldOptions) (runner.Runner, []*latest.SkaffoldConfig, error) {
	runCtx, configs, err := runContext(opts)
	if err != nil {
		return nil, nil, err
	}

	instrumentation.InitMeterFromConfig(configs)
	runner, err := runner.NewForConfig(runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, fmt.Errorf("creating runner: %w", err)
	}

	return runner, configs, nil
}

func runContext(opts config.SkaffoldOptions) (*runcontext.RunContext, []*latest.SkaffoldConfig, error) {
	parsed, err := schema.ParseConfigAndUpgrade(opts.ConfigurationFile, latest.Version)
	if err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return nil, nil, fmt.Errorf("skaffold config file %s not found - check your current working directory, or try running `skaffold init`", opts.ConfigurationFile)
		}

		// If the error is NOT that the file doesn't exist, then we warn the user
		// that maybe they are using an outdated version of Skaffold that's unable to read
		// the configuration.
		warnIfUpdateIsAvailable()
		return nil, nil, fmt.Errorf("parsing skaffold config: %w", err)
	}

	if len(parsed) == 0 {
		return nil, nil, fmt.Errorf("skaffold config file %s is empty", opts.ConfigurationFile)
	}

	setDefaultDeployer := setDefaultDeployer(parsed)
	var pipelines []latest.Pipeline
	var configs []*latest.SkaffoldConfig
	for _, cfg := range parsed {
		config := cfg.(*latest.SkaffoldConfig)

		if err = schema.ApplyProfiles(config, opts); err != nil {
			return nil, nil, fmt.Errorf("applying profiles: %w", err)
		}
		if err := defaults.Set(config, setDefaultDeployer); err != nil {
			return nil, nil, fmt.Errorf("setting default values: %w", err)
		}
		pipelines = append(pipelines, config.Pipeline)
		configs = append(configs, config)
	}

	// TODO: Should support per-config kubecontext. Right now we constrain all configs to define the same kubecontext.
	kubectx.ConfigureKubeConfig(opts.KubeConfig, opts.KubeContext, configs[0].Deploy.KubeContext)

	if err := validation.Process(configs); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	runCtx, err := runcontext.GetRunContext(opts, pipelines)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	if err := validation.ProcessWithRunContext(runCtx); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	return runCtx, configs, nil
}

func setDefaultDeployer(configs []util.VersionedConfig) bool {
	// set the default deployer only if no deployer is explicitly specified in any config
	for _, cfg := range configs {
		if cfg.(*latest.SkaffoldConfig).Deploy.DeployType != (latest.DeployType{}) {
			return false
		}
	}
	return true
}

func warnIfUpdateIsAvailable() {
	warning, err := update.CheckVersionOnError(opts.GlobalConfig)
	if err != nil {
		logrus.Infof("update check failed: %s", err)
		return
	}
	if warning != "" {
		logrus.Warn(warning)
	}
}
