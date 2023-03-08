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
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockDevRunner struct {
	runner.Runner
	hasBuilt          bool
	errDev            error
	calls             []string
	deployedManifests manifest.ManifestListByConfig
}

func (r *mockDevRunner) Dev(context.Context, io.Writer, []*latest.Artifact) error {
	r.calls = append(r.calls, "Dev")
	return r.errDev
}

func (r *mockDevRunner) HasBuilt() bool {
	r.calls = append(r.calls, "HasBuilt")
	return r.hasBuilt
}

func (r *mockDevRunner) DeployManifests() manifest.ManifestListByConfig {
	r.calls = append(r.calls, "DeployManifests")
	return r.deployedManifests
}

func (r *mockDevRunner) Prune(context.Context, io.Writer) error {
	r.calls = append(r.calls, "Prune")
	return nil
}

func (r *mockDevRunner) Cleanup(context.Context, io.Writer, bool, manifest.ManifestListByConfig, string) error {
	r.calls = append(r.calls, "Cleanup")
	return nil
}

func (r *mockDevRunner) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	r.calls = append(r.calls, "Render")
	return r.deployedManifests, nil
}

func TestDoDev(t *testing.T) {
	tests := []struct {
		description       string
		hasBuilt          bool
		deployedManifests manifest.ManifestList
		expectedCalls     []string
	}{
		{
			description:       "cleanup and then prune",
			hasBuilt:          true,
			deployedManifests: manifest.ManifestList{[]byte("dummy")},
			expectedCalls:     []string{"Dev", "DeployManifests", "HasBuilt", "Cleanup", "Prune"},
		},
		{
			description:       "cleanup always gets called even without deployments",
			hasBuilt:          true,
			deployedManifests: manifest.ManifestList{},
			expectedCalls:     []string{"Dev", "DeployManifests", "HasBuilt", "Cleanup", "Prune"},
		},
		{
			description:       "hasn't built",
			hasBuilt:          false,
			deployedManifests: manifest.ManifestList{},
			expectedCalls:     []string{"Dev", "DeployManifests", "HasBuilt", "Cleanup"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dm := manifest.NewManifestListByConfig()
			dm.Add("test", test.deployedManifests)
			mockRunner := &mockDevRunner{
				hasBuilt:          test.hasBuilt,
				deployedManifests: dm,
				errDev:            context.Canceled,
			}
			t.Override(&createRunner, func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
				return mockRunner, []util.VersionedConfig{&latest.SkaffoldConfig{}}, nil, nil
			})
			t.Override(&opts, config.SkaffoldOptions{
				Cleanup: true,
				NoPrune: false,
			})

			err := doDev(context.Background(), io.Discard)

			t.CheckDeepEqual(test.expectedCalls, mockRunner.calls)
			t.CheckTrue(err == context.Canceled)
		})
	}
}

type mockConfigChangeRunner struct {
	runner.Runner
	cycles int
}

func (m *mockConfigChangeRunner) Dev(context.Context, io.Writer, []*latest.Artifact) error {
	m.cycles++
	if m.cycles == 1 {
		// pass through the first cycle with a config reload
		return runner.ErrorConfigurationChanged
	}
	return context.Canceled
}

func (m *mockConfigChangeRunner) HasBuilt() bool {
	return true
}

func (m *mockConfigChangeRunner) DeployManifests() manifest.ManifestListByConfig {
	return manifest.ManifestListByConfig{}
}

func (m *mockConfigChangeRunner) Prune(context.Context, io.Writer) error {
	return nil
}

func (m *mockConfigChangeRunner) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	return manifest.ManifestListByConfig{}, nil
}

func (m *mockConfigChangeRunner) Cleanup(context.Context, io.Writer, bool, manifest.ManifestListByConfig, string) error {
	return nil
}

func TestDevConfigChange(t *testing.T) {
	testutil.Run(t, "test config change", func(t *testutil.T) {
		mockRunner := &mockConfigChangeRunner{}

		t.Override(&createRunner, func(context.Context, io.Writer, config.SkaffoldOptions) (runner.Runner, []util.VersionedConfig, *runcontext.RunContext, error) {
			return mockRunner, []util.VersionedConfig{&latest.SkaffoldConfig{}}, nil, nil
		})
		t.Override(&opts, config.SkaffoldOptions{
			Cleanup: true,
			NoPrune: false,
		})

		err := doDev(context.Background(), io.Discard)

		// ensure that we received the context.Canceled error (and not ErrorConfigurationChanged)
		// also ensure that the we run through dev cycles (since we reloaded on the first),
		// and exit after a real error is received
		t.CheckTrue(err == context.Canceled)
		t.CheckDeepEqual(mockRunner.cycles, 2)
	})
}

func TestNewCmdDev(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Chdir()
		t.Override(&opts, config.SkaffoldOptions{})

		cmd := NewCmdDev()
		cmd.SilenceUsage = true
		cmd.Execute()

		t.CheckDeepEqual(true, opts.Tail)
		t.CheckDeepEqual(false, opts.Force)
	})
}
