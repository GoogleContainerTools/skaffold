/*
Copyright 2023 The Skaffold Authors

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

package runner

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/actions/k8sjob"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// ActionsRunner defines the API used to run custom actions.
type ActionsRunner interface {
	// Exec triggers the execution of the given action.
	Exec(ctx context.Context, out io.Writer, allbuilds []graph.Artifact, localImgs []graph.Artifact, action string) error

	// ExecAll triggers the execution of all the defined actions.
	ExecAll(ctx context.Context, out io.Writer, allbuilds, localImgs []graph.Artifact) error
}

func GetActionsRunner(ctx context.Context, runCtx *runcontext.RunContext, l *label.DefaultLabeller, dockerNetwork string, envFile string) (ActionsRunner, error) {
	aCfgs := []latest.Action{}

	for _, p := range runCtx.GetPipelines() {
		aCfgs = append(aCfgs, p.CustomActions...)
	}

	envMap, err := loadEnvMap(envFile)
	if err != nil {
		return nil, err
	}

	return createActionsRunner(ctx, runCtx, l, dockerNetwork, envMap, aCfgs)
}

func createActionsRunner(ctx context.Context, runCtx *runcontext.RunContext, l *label.DefaultLabeller, dNetwork string, envMap map[string]string, aCfgs []latest.Action) (ActionsRunner, error) {
	execEnvByAction := map[string]actions.ExecEnv{}
	ordExecEnvs := []actions.ExecEnv{}
	acsByExecEnv := map[actions.ExecEnv][]string{}

	pF := runCtx.PortForwardResources()
	dockerCfgs, k8sCfgs := cfgsByExecMode(aCfgs)

	if len(dockerCfgs) > 0 {
		dExecEnv, err := docker.NewExecEnv(ctx, runCtx, l, pF, dNetwork, envMap, dockerCfgs)
		if err != nil {
			return nil, err
		}
		ordExecEnvs = append(ordExecEnvs, dExecEnv)
		insertExecEnv(dExecEnv, dockerCfgs, execEnvByAction, acsByExecEnv)
	}

	if len(k8sCfgs) > 0 {
		kExecEnv := k8sjob.NewExecEnv(ctx, runCtx, l, runCtx.GetNamespace(), envMap, k8sCfgs)
		ordExecEnvs = append(ordExecEnvs, kExecEnv)
		insertExecEnv(kExecEnv, k8sCfgs, execEnvByAction, acsByExecEnv)
	}

	return actions.NewRunner(execEnvByAction, ordExecEnvs, acsByExecEnv), nil
}

func insertExecEnv(execEnv actions.ExecEnv, acs []latest.Action, execEnvByAction map[string]actions.ExecEnv, acsByExecEnv map[actions.ExecEnv][]string) {
	for _, a := range acs {
		acsByExecEnv[execEnv] = append(acsByExecEnv[execEnv], a.Name)
		execEnvByAction[a.Name] = execEnv
	}
}

func cfgsByExecMode(aCfgs []latest.Action) (dockerCfgs []latest.Action, k8sCfgs []latest.Action) {
	for _, cfg := range aCfgs {
		setDefaultConfigValues(&cfg)
		if cfg.ExecutionModeConfig.KubernetesClusterExecutionMode != nil {
			k8sCfgs = append(k8sCfgs, cfg)
			continue
		}
		dockerCfgs = append(dockerCfgs, cfg)
	}
	return
}

func setDefaultConfigValues(cfg *latest.Action) {
	if cfg.Config.IsFailFast == nil {
		cfg.Config.IsFailFast = util.Ptr(true)
	}

	if cfg.Config.Timeout == nil {
		cfg.Config.Timeout = util.Ptr(0) // No timeout
	}
}

func loadEnvMap(envFile string) (map[string]string, error) {
	if envFile == "" {
		return nil, nil
	}
	return util.ParseEnvVariablesFromFile(envFile)
}
