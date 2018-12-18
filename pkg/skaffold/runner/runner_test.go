/*
Copyright 2018 The Skaffold Authors

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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/local"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type TestBuilder struct {
	built  [][]string
	errors []error
	tag    int
}

func (t *TestBuilder) Labels() map[string]string {
	return map[string]string{}
}

func (t *TestBuilder) Build(ctx context.Context, w io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.tag++

	var builds []build.Artifact
	for _, artifact := range artifacts {
		builds = append(builds, build.Artifact{
			ImageName: artifact.ImageName,
			Tag:       fmt.Sprintf("%s:%d", artifact.ImageName, t.tag),
		})
	}

	t.built = append(t.built, tags(builds))
	return builds, nil
}

type TestTester struct {
	tested [][]string
	errors []error
}

func (t *TestTester) Test(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		if err != nil {
			return err
		}
	}

	t.tested = append(t.tested, tags(artifacts))
	return nil
}

func (t *TestTester) TestDependencies() ([]string, error) {
	return nil, nil
}

type TestDeployer struct {
	deployed [][]string
	errors   []error
}

func (t *TestDeployer) Labels() map[string]string {
	return map[string]string{}
}

func (t *TestDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (t *TestDeployer) Deploy(ctx context.Context, out io.Writer, artifacts []build.Artifact) ([]deploy.Artifact, error) {
	if len(t.errors) > 0 {
		err := t.errors[0]
		t.errors = t.errors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.deployed = append(t.deployed, tags(artifacts))
	return nil, nil
}

func (t *TestDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func tags(artifacts []build.Artifact) []string {
	var tags []string
	for _, artifact := range artifacts {
		tags = append(tags, artifact.Tag)
	}
	return tags
}

func createDefaultRunner(t *testing.T) *SkaffoldRunner {
	t.Helper()

	opts := &config.SkaffoldOptions{
		Trigger: "polling",
	}

	pipeline := &latest.SkaffoldPipeline{}
	defaults.Set(pipeline)

	runner, err := NewForConfig(opts, pipeline)

	testutil.CheckError(t, false, err)

	return runner
}

func TestNewForConfig(t *testing.T) {
	restore := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: "cluster1"})
	defer restore()

	var tests = []struct {
		description      string
		pipeline         *latest.SkaffoldPipeline
		shouldErr        bool
		expectedBuilder  build.Builder
		expectedTester   test.Tester
		expectedDeployer deploy.Deployer
	}{
		{
			description: "local builder config",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			},
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "bad tagger config",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
				Deploy: latest.DeployConfig{
					DeployType: latest.DeployType{
						KubectlDeploy: &latest.KubectlDeploy{},
					},
				},
			},
			shouldErr: true,
		},
		{
			description: "unknown builder",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{},
			},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown tagger",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				}},
			shouldErr:        true,
			expectedBuilder:  &local.Builder{},
			expectedTester:   &test.FullTester{},
			expectedDeployer: &deploy.KubectlDeployer{},
		},
		{
			description: "unknown deployer",
			pipeline: &latest.SkaffoldPipeline{
				Build: latest.BuildConfig{
					TagPolicy: latest.TagPolicy{ShaTagger: &latest.ShaTagger{}},
					BuildType: latest.BuildType{
						LocalBuild: &latest.LocalBuild{},
					},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := NewForConfig(&config.SkaffoldOptions{
				Trigger: "polling",
			}, test.pipeline)

			testutil.CheckError(t, test.shouldErr, err)
			if cfg != nil {
				b, _t, d := WithTimings(test.expectedBuilder, test.expectedTester, test.expectedDeployer)

				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, b, cfg.Builder)
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, _t, cfg.Tester)
				testutil.CheckErrorAndTypeEquality(t, test.shouldErr, err, d, cfg.Deployer)
			}
		})
	}
}

func TestRun(t *testing.T) {
	var tests = []struct {
		description string
		builder     build.Builder
		tester      test.Tester
		deployer    deploy.Deployer
		shouldErr   bool
	}{
		{
			description: "run no error",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{},
		},
		{
			description: "run build error",
			builder:     &TestBuilder{errors: []error{errors.New("")}},
			tester:      &TestTester{},
			shouldErr:   true,
		},
		{
			description: "run deploy error",
			builder:     &TestBuilder{},
			tester:      &TestTester{},
			deployer:    &TestDeployer{errors: []error{errors.New("")}},
			shouldErr:   true,
		},
		{
			description: "run test error",
			builder:     &TestBuilder{},
			tester:      &TestTester{errors: []error{errors.New("")}},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			runner := createDefaultRunner(t)
			runner.Builder = test.builder
			runner.Tester = test.tester
			runner.Deployer = test.deployer

			err := runner.Run(context.Background(), ioutil.Discard, []*latest.Artifact{{
				ImageName: "test",
			}})

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
