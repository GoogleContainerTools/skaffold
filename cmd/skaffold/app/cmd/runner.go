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
	"io"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer"
	initConfig "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	runcontext "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext/v2"
	v2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/defaults"
	latestV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// For tests
var createRunner = createNewRunner

func withRunner(ctx context.Context, out io.Writer, action func(runner.Runner, []*latestV2.SkaffoldConfig) error) error {
	runner, config, runCtx, err := createRunner(out, opts)
	if err != nil {
		return err
	}

	err = action(runner, config)

	return alwaysSucceedWhenCancelled(ctx, runCtx, err)
}

// createNewRunner creates a Runner and returns the SkaffoldConfig associated with it.
func createNewRunner(out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []*latestV2.SkaffoldConfig, *runcontext.RunContext, error) {
	runCtx, configs, err := runContext(out, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	instrumentation.Init(configs, opts.User)
	runner, err := v2.NewForConfig(runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, nil, fmt.Errorf("creating runner: %w", err)
	}
	return runner, configs, runCtx, nil
}

func runContext(out io.Writer, opts config.SkaffoldOptions) (*runcontext.RunContext, []*latestV2.SkaffoldConfig, error) {
	cfgSet, err := withFallbackConfig(out, opts, parser.GetConfigSet)
	if err != nil {
		return nil, nil, err
	}

	if err := validation.Process(cfgSet, validation.GetValidationOpts(opts)); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}
	var configs []*latestV2.SkaffoldConfig
	for _, cfg := range cfgSet {
		configs = append(configs, cfg.SkaffoldConfig)
	}
	runCtx, err := runcontext.GetRunContext(opts, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	if err := validation.ProcessWithRunContext(runCtx); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	return runCtx, configs, nil
}

// withFallbackConfig will try to automatically generate a config if root `skaffold.yaml` file does not exist.
func withFallbackConfig(out io.Writer, opts config.SkaffoldOptions, getCfgs func(opts config.SkaffoldOptions) (parser.SkaffoldConfigSet, error)) (parser.SkaffoldConfigSet, error) {
	configs, err := getCfgs(opts)
	if err == nil {
		// do not set a default deployer in a multi-config application.
		if len(configs) == 1 {
			defaults.SetDefaultRenderer(configs[0].SkaffoldConfig)
			defaults.SetDefaultDeployer(configs[0].SkaffoldConfig)
		}
		return configs, nil
	}
	var e sErrors.Error
	if errors.As(err, &e) && e.StatusCode() == proto.StatusCode_CONFIG_FILE_NOT_FOUND_ERR {
		if opts.AutoCreateConfig && initializer.ValidCmd(opts) {
			output.Default.Fprintf(out, "Skaffold config file %s not found - Trying to create one for you...\n", opts.ConfigurationFile)
			config, err := initializer.Transparent(context.Background(), out, initConfig.Config{Opts: opts})
			if err != nil {
				return nil, fmt.Errorf("unable to generate skaffold config file automatically - try running `skaffold init`: %w", err)
			}
			if config == nil {
				return nil, fmt.Errorf("unable to generate skaffold config file automatically - try running `skaffold init`: action cancelled by user")
			}

			defaults.Set(config)

			return parser.SkaffoldConfigSet{
				&parser.SkaffoldConfigEntry{SkaffoldConfig: config, IsRootConfig: true},
			}, nil
		}

		return nil, fmt.Errorf("skaffold config file %s not found - check your current working directory, or try running `skaffold init`", opts.ConfigurationFile)
	}

	// If the error is NOT that the file doesn't exist, then we warn the user
	// that maybe they are using an outdated version of Skaffold that's unable to read
	// the configuration.
	warnIfUpdateIsAvailable()
	return nil, fmt.Errorf("parsing skaffold config: %w", err)
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
