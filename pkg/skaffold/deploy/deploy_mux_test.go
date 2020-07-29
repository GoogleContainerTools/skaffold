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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func NewMockDeployer() *MockDeployer { return &MockDeployer{labels: make(map[string]string)} }

type MockDeployer struct {
	labels          map[string]string
	deployResources kubectl.Resources
	deployErr       error
	dependencies    []string
	dependenciesErr error
	cleanupErr      error
	renderResult    string
	renderErr       error
}

func (m *MockDeployer) Dependencies() ([]string, error) {
	return m.dependencies, m.dependenciesErr
}

func (m *MockDeployer) Cleanup(context.Context, io.Writer) error {
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

func (m *MockDeployer) Deploy(context.Context, io.Writer, []build.Artifact) (kubectl.Resources, error) {
	return m.deployResources, m.deployErr
}

func (m *MockDeployer) Render(_ context.Context, w io.Writer, _ []build.Artifact, _ bool, _ string) error {
	w.Write([]byte(m.renderResult))
	return m.renderErr
}

func (m *MockDeployer) WithDeployResources(deployResources kubectl.Resources) *MockDeployer {
	m.deployResources = deployResources
	return m
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
		name              string
		resources1        kubectl.Resources
		resources2        kubectl.Resources
		err1              error
		err2              error
		expectedResources kubectl.Resources
		shouldErr         bool
	}{
		{
			name:              "combine resources",
			resources1:        []kubectl.Resource{{Name: "a"}},
			resources2:        []kubectl.Resource{{Name: "b"}},
			expectedResources: []kubectl.Resource{{Name: "a"}, {Name: "b"}},
		},
		{
			name:              "short-circuits when first call fails",
			resources1:        []kubectl.Resource{{Name: "a"}},
			err1:              errors.New("failed in first"),
			resources2:        []kubectl.Resource{{Name: "b"}},
			expectedResources: nil,
			shouldErr:         true,
		},
		{
			name:              "when second call fails",
			resources1:        []kubectl.Resource{{Name: "a"}},
			resources2:        []kubectl.Resource{{Name: "b"}},
			err2:              errors.New("failed in second"),
			expectedResources: nil,
			shouldErr:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployerMux := DeployerMux([]Deployer{
				NewMockDeployer().WithDeployResources(test.resources1).WithDeployErr(test.err1),
				NewMockDeployer().WithDeployResources(test.resources2).WithDeployErr(test.err2),
			})

			resources, err := deployerMux.Deploy(context.Background(), nil, nil)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedResources, resources)
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
			deps1:        []string{"dep-a"},
			deps2:        []string{"dep-b"},
			expectedDeps: []string{"dep-a", "dep-b"},
		},
		{
			name:         "repeated dependencies are not duplicated",
			deps1:        []string{"dep-a", "dep-c"},
			deps2:        []string{"dep-b", "dep-c"},
			expectedDeps: []string{"dep-a", "dep-b", "dep-c"},
		},
		{
			name:      "when first call fails",
			deps1:     []string{"dep-a"},
			err1:      fmt.Errorf("failed in first"),
			deps2:     []string{"dep-b"},
			shouldErr: true,
		},
		{
			name:      "when second call fails",
			deps1:     []string{"dep-a"},
			deps2:     []string{"dep-b"},
			err2:      fmt.Errorf("failed in second"),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployerMux := DeployerMux([]Deployer{
				NewMockDeployer().WithDependencies(test.deps1).WithDependenciesErr(test.err1),
				NewMockDeployer().WithDependencies(test.deps2).WithDependenciesErr(test.err2),
			})

			dependencies, err := deployerMux.Dependencies()
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedDeps, dependencies)
		})
	}
}

func TestDeployerMux_Render(t *testing.T) {
	tests := []struct {
		name           string
		render1        string
		render2        string
		err1           error
		err2           error
		expectedRender string
		shouldErr      bool
	}{
		{
			name:           "concatenates render results with separator",
			render1:        "manifest-1",
			render2:        "manifest-2",
			expectedRender: "manifest-1\n---\nmanifest-2\n",
		},
		{
			name:      "short-circuits when first call fails",
			render1:   "manifest-1",
			err1:      fmt.Errorf("failed in first"),
			render2:   "manifest-2",
			shouldErr: true,
		},
		{
			name:      "short-circuits when second call fails",
			render1:   "manifest-1",
			render2:   "manifest-2",
			err2:      fmt.Errorf("failed in first"),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run("output to writer "+test.name, func(t *testing.T) {
			deployerMux := DeployerMux([]Deployer{
				NewMockDeployer().WithRenderResult(test.render1).WithRenderErr(test.err1),
				NewMockDeployer().WithRenderResult(test.render2).WithRenderErr(test.err2),
			})

			buf := &bytes.Buffer{}
			err := deployerMux.Render(context.Background(), buf, nil, true, "")
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedRender, buf.String())
		})
	}

	t.Run("output to file", func(t *testing.T) {
		// only check the good case here
		test := tests[0]

		tmpDir := testutil.NewTempDir(t)

		deployerMux := DeployerMux([]Deployer{
			NewMockDeployer().WithRenderResult(test.render1).WithRenderErr(test.err1),
			NewMockDeployer().WithRenderResult(test.render2).WithRenderErr(test.err2),
		})

		err := deployerMux.Render(context.Background(), nil, nil, true, tmpDir.Path("render"))
		testutil.CheckError(t, false, err)

		file, _ := os.Open(tmpDir.Path("render"))
		content, _ := ioutil.ReadAll(file)
		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedRender, string(content))
	})
}
