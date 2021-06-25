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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer"
	initConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	runcontextv1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v1"
	runcontextv2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/v1"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// For tests
var createRunner = createNewRunnerV1
var parseAllConfigs = withFallbackConfig

func withRunner(ctx context.Context, out io.Writer, action func(runner.Runner) error) error {
	configs, err := parseAllConfigs(out, opts, parser.GetAllConfigs)
	if err != nil {
		return err
	}
	var runner runner.Runner
	var runCtx interface{}
	if _, found := schema.SchemaVersionsV2.Find(configs[0].GetVersion()); found {
		runner, runCtx, err = createNewRunnerV2(opts, configs)
		if err != nil {
			return err
		}
	} else {
		runner, runCtx, err = createRunner(opts, configs)
		if err != nil {
			return err
		}
	}
	if err := action(runner); err != nil {
		// if the context was cancelled act as if all is well
		if ctx.Err() == context.Canceled {
			return nil
		} else if err == context.Canceled {
			return err
		}
		return sErrors.ShowAIError(runCtx, err)
	}
	return nil
}

// createNewRunnerV1 creates a v1 Runner and returns the v1 SkaffoldConfig associated with it.
func createNewRunnerV1(opts config.SkaffoldOptions, configs []util.VersionedConfig) (runner.Runner, *runcontextv1.RunContext, error) {
	runCtx, err := runcontextv1.GetRunContext(opts, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}
	var v1Configs []*latestV1.SkaffoldConfig
	for _, c := range configs {
		v1Configs = append(v1Configs, c.(*latestV1.SkaffoldConfig))
	}
	setDefaultDeployer(v1Configs)
	if err := validation.Process(v1Configs); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}
	if err := validation.ProcessWithRunContext(runCtx); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}
	instrumentation.Init(v1Configs, opts.User)
	runner, err := v1.NewForConfig(runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, fmt.Errorf("creating runner: %w", err)
	}
	runner.SetV1Config(v1Configs)
	return runner, runCtx, nil
}

// createNewRunnerV2 creates a v2 Runner and returns the v2 SkaffoldConfig associated with it.
func createNewRunnerV2(opts config.SkaffoldOptions, configs []util.VersionedConfig) (runner.Runner, *runcontextv2.RunContext, error) {
	var v2Configs []*latestV2.SkaffoldConfig
	for _, c := range configs {
		v2Configs = append(v2Configs, c.(*latestV2.SkaffoldConfig))
	}
	// TODO(yuwenma): set default deploy for v2
	// TODO: validate for v2
	// TODO: instrumentation.Init
	runCtx, err := runcontextv2.GetRunContext(opts, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	runner, err := v2.NewForConfig(runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, fmt.Errorf("creating runner: %w", err)
	}
	runner.SetV2Config(v2Configs)
	return runner, runCtx, nil
}

// withFallbackConfig will try to automatically generate a config if root `skaffold.yaml` file does not exist.
func withFallbackConfig(out io.Writer, opts config.SkaffoldOptions, getCfgs func(opts config.SkaffoldOptions) ([]util.VersionedConfig, error)) ([]util.VersionedConfig, error) {
	configs, err := getCfgs(opts)
	if err == nil {
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

			return []util.VersionedConfig{config}, nil
		}

		return nil, fmt.Errorf("skaffold config file %s not found - check your current working directory, or try running `skaffold init`", opts.ConfigurationFile)
	}

	// If the error is NOT that the file doesn't exist, then we warn the user
	// that maybe they are using an outdated version of Skaffold that's unable to read
	// the configuration.
	warnIfUpdateIsAvailable()
	return nil, fmt.Errorf("parsing skaffold config: %w", err)
}

func setDefaultDeployer(configs []*latestV1.SkaffoldConfig) {
	// do not set a default deployer in a multi-config application.
	if len(configs) > 1 {
		return
	}
	// there always exists at least one config
	// TODO: yuwen determine default deployer for v2
	defaults.SetDefaultDeployer(configs[0])
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
