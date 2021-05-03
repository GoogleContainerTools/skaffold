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
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func MockRunnerV1(t *testutil.T, testBench *test.TestBench, monitor filemon.Monitor, artifacts []*latest_v1.Artifact,
	autoTriggers *test.TriggerState) *SkaffoldRunner {
	if autoTriggers == nil {
		autoTriggers = &test.TriggerState{true, true, true}
	}
	cfg := &latest_v1.SkaffoldConfig{
		Pipeline: latest_v1.Pipeline{
			Build: latest_v1.BuildConfig{
				TagPolicy: latest_v1.TagPolicy{
					// Use the fastest tagger
					ShaTagger: &latest_v1.ShaTagger{},
				},
				Artifacts: artifacts,
			},
			Deploy: latest_v1.DeployConfig{StatusCheckDeadlineSeconds: 60},
		},
	}
	defaults.Set(cfg)
	defaults.SetDefaultDeployer(cfg)
	runCtx := &runcontext.RunContext{
		Pipelines: runcontext.NewPipelines([]latest_v1.Pipeline{cfg.Pipeline}),
		Opts: config.SkaffoldOptions{
			Trigger:           "polling",
			WatchPollInterval: 100,
			AutoBuild:         autoTriggers.Build,
			AutoSync:          autoTriggers.Sync,
			AutoDeploy:        autoTriggers.Deploy,
		},
	}
	newConfig, err := NewForConfig(runCtx)
	t.CheckNoError(err)
	runnerV1 := newConfig.(*SkaffoldRunner)
	runnerV1.Builder.Builder = testBench
	runnerV1.Syncer = testBench
	runnerV1.Tester.Tester = testBench
	runnerV1.Deployer = testBench
	runnerV1.Listener = testBench
	runnerV1.Monitor = monitor

	testBench.DevLoop = func(ctx context.Context, out io.Writer, doDev func() error) error {
		if err := monitor.Run(true); err != nil {
			return err
		}
		return doDev()
	}

	testBench.FirstMonitor = func(bool) error {
		// default to noop so we don't add extra actions
		return nil
	}

	return runnerV1
}
