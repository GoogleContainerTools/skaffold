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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer"
	initConfig "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/validation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// For tests
var createRunner = createNewRunner

func withRunner(ctx context.Context, out io.Writer, action func(runner.Runner, []util.VersionedConfig) error) error {
	runner, config, runCtx, err := createRunner(ctx, out, opts)
	if err != nil {
		return err
	}

	err = action(runner, config)

	return alwaysSucceedWhenCancelled(ctx, runCtx, err)
}

// createNewRunner creates a Runner and returns the SkaffoldConfig associated with it.
func createNewRunner(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
	runCtx, configs, err := runContext(ctx, out, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	var v2Configs []*latest.SkaffoldConfig
	for _, c := range configs {
		v2Configs = append(v2Configs, c.(*latest.SkaffoldConfig))
	}
	instrumentation.Init(v2Configs, opts.User, runCtx.GetKubeContext())
	hooks.SetupStaticEnvOptions(runCtx)
	runner, err := runner.NewForConfig(ctx, runCtx)
	if err != nil {
		event.InititializationFailed(err)
		return nil, nil, nil, fmt.Errorf("creating runner: %w", err)
	}
	return runner, configs, runCtx, nil
}

func runContext(ctx context.Context, out io.Writer, opts config.SkaffoldOptions) (*runcontext.RunContext, []util.VersionedConfig, error) {
	cfgSet, err := withFallbackConfig(ctx, out, opts, parser.GetConfigSet)
	if err != nil {
		return nil, nil, err
	}
	setDefaultRendererAndDeployer(cfgSet)

	if err := validation.Process(cfgSet, validation.GetValidationOpts(opts)); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}
	var configs []util.VersionedConfig
	for _, cfg := range cfgSet {
		configs = append(configs, cfg.SkaffoldConfig)
	}

	runCtx, err := runcontext.GetRunContext(ctx, opts, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("getting run context: %w", err)
	}

	if err := validation.ProcessWithRunContext(ctx, runCtx); err != nil {
		return nil, nil, fmt.Errorf("invalid skaffold config: %w", err)
	}

	return runCtx, configs, nil
}

// withFallbackConfig will try to automatically generate a config if root `skaffold.yaml` file does not exist.
func withFallbackConfig(ctx context.Context, out io.Writer, opts config.SkaffoldOptions, getCfgs func(context.Context, config.SkaffoldOptions) (parser.SkaffoldConfigSet, error)) (parser.SkaffoldConfigSet, error) {
	configs, err := getCfgs(ctx, opts)
	if err == nil {
		return configs, nil
	}
	var e sErrors.Error
	if errors.As(err, &e) && e.StatusCode() == proto.StatusCode_CONFIG_FILE_NOT_FOUND_ERR {
		if (opts.AutoCreateConfig || opts.AutoInit) && initializer.ValidCmd(opts) {
			output.Default.Fprintf(out, "Skaffold config file %s not found - Trying to create one for you...\n", opts.ConfigurationFile)
			config, err := initializer.Transparent(context.Background(), out, initConfig.Config{Opts: opts, EnableManifestGeneration: opts.AutoInit})
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
		log.Entry(context.TODO()).Infof("update check failed: %s", err)
		return
	}
	if warning != "" {
		log.Entry(context.TODO()).Warn(warning)
	}
}

func setDefaultRendererAndDeployer(configs parser.SkaffoldConfigSet) {
	// do not set a default deployer or renderer in a multi-config application.
	if len(configs) > 1 {
		return
	}
	// there always exists at least one config
	defaults.SetDefaultRenderer(configs[0].SkaffoldConfig)
	defaults.SetDefaultDeployer(configs[0].SkaffoldConfig)
}
