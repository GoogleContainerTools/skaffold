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

package deploy

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/testutil/event"
)

func NewMockDeployer() *MockDeployer { return &MockDeployer{labels: make(map[string]string)} }

type MockDeployer struct {
	labels          map[string]string
	deployErr       error
	dependencies    []string
	dependenciesErr error
	cleanupErr      error
	renderResult    string
	renderErr       error
}

func (m *MockDeployer) HasRunnableHooks() bool {
	return true
}

func (m *MockDeployer) PreDeployHooks(context.Context, io.Writer) error {
	return nil
}

func (m *MockDeployer) PostDeployHooks(context.Context, io.Writer) error {
	return nil
}

func (m *MockDeployer) GetAccessor() access.Accessor {
	return &access.NoopAccessor{}
}

func (m *MockDeployer) GetDebugger() debug.Debugger {
	return &debug.NoopDebugger{}
}

func (m *MockDeployer) GetLogger() log.Logger {
	return &log.NoopLogger{}
}

func (m *MockDeployer) GetStatusMonitor() status.Monitor {
	return &status.NoopMonitor{}
}

func (m *MockDeployer) GetSyncer() sync.Syncer {
	return &sync.NoopSyncer{}
}

func (m *MockDeployer) RegisterLocalImages(_ []graph.Artifact) {}

func (m *MockDeployer) TrackBuildArtifacts(_ []graph.Artifact) {}

func (m *MockDeployer) Dependencies() ([]string, error) {
	return m.dependencies, m.dependenciesErr
}

func (m *MockDeployer) Cleanup(context.Context, io.Writer, bool, manifest.ManifestList) error {
	return m.cleanupErr
}

func (m *MockDeployer) WithLabel(labels map[string]string) *MockDeployer {
	m.labels = labels
	return m
}

func (m *MockDeployer) WithDeployErr(err error) *MockDeployer {
	m.deployErr = err
	return m
}

func (m *MockDeployer) WithDependenciesErr(err error) *MockDeployer {
	m.dependenciesErr = err
	return m
}

func (m *MockDeployer) WithCleanupErr(err error) *MockDeployer {
	m.cleanupErr = err
	return m
}

func (m *MockDeployer) WithRenderErr(err error) *MockDeployer {
	m.renderErr = err
	return m
}

func (m *MockDeployer) Deploy(context.Context, io.Writer, []graph.Artifact, manifest.ManifestList) error {
	return m.deployErr
}

func (m *MockDeployer) Render(_ context.Context, w io.Writer, _ []graph.Artifact, _ bool, _ string) error {
	w.Write([]byte(m.renderResult))
	return m.renderErr
}

func (m *MockDeployer) WithDependencies(dependencies []string) *MockDeployer {
	m.dependencies = dependencies
	return m
}

func (m *MockDeployer) WithRenderResult(renderResult string) *MockDeployer {
	m.renderResult = renderResult
	return m
}

func TestDeployerMux_Deploy(t *testing.T) {
	tests := []struct {
		name        string
		namespaces1 []string
		namespaces2 []string
		err1        error
		err2        error
		expectedNs  []string
		shouldErr   bool
	}{
		{
			name:      "short-circuits when first call fails",
			err1:      fmt.Errorf("failed in first"),
			shouldErr: true,
		},
		{
			name:      "when second call fails",
			err2:      fmt.Errorf("failed in second"),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEvent.InitializeState([]latest.Pipeline{{
				Deploy: latest.DeployConfig{},
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}}})

			deployerMux := NewDeployerMux([]Deployer{
				NewMockDeployer().WithDeployErr(test.err1),
				NewMockDeployer().WithDeployErr(test.err2),
			}, false)

			err := deployerMux.Deploy(context.Background(), nil, nil, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestDeployerMux_Dependencies(t *testing.T) {
	tests := []struct {
		name         string
		deps1        []string
		deps2        []string
		err1         error
		err2         error
		expectedDeps []string
		shouldErr    bool
	}{
		{
			name:         "disjoint dependencies are combined",
			deps1:        []string{"graph-a"},
			deps2:        []string{"graph-b"},
			expectedDeps: []string{"graph-a", "graph-b"},
		},
		{
			name:         "repeated dependencies are not duplicated",
			deps1:        []string{"graph-a", "graph-c"},
			deps2:        []string{"graph-b", "graph-c"},
			expectedDeps: []string{"graph-a", "graph-b", "graph-c"},
		},
		{
			name:      "when first call fails",
			deps1:     []string{"graph-a"},
			err1:      fmt.Errorf("failed in first"),
			deps2:     []string{"graph-b"},
			shouldErr: true,
		},
		{
			name:      "when second call fails",
			deps1:     []string{"graph-a"},
			deps2:     []string{"graph-b"},
			err2:      fmt.Errorf("failed in second"),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployerMux := NewDeployerMux([]Deployer{
				NewMockDeployer().WithDependencies(test.deps1).WithDependenciesErr(test.err1),
				NewMockDeployer().WithDependencies(test.deps2).WithDependenciesErr(test.err2),
			}, false)

			dependencies, err := deployerMux.Dependencies()
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedDeps, dependencies)
		})
	}
}
