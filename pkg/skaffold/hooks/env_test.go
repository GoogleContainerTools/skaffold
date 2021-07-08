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

package hooks

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSetupStaticEnvOptions(t *testing.T) {
	defer func() {
		staticEnvOpts = StaticEnvOpts{}
	}()

	cfg := mockCfg{
		defaultRepo: util.StringPtr("gcr.io/foo"),
		workDir:     ".",
		rpcPort:     8080,
		httpPort:    8081,
	}
	SetupStaticEnvOptions(cfg)
	testutil.CheckDeepEqual(t, cfg.defaultRepo, staticEnvOpts.DefaultRepo)
	testutil.CheckDeepEqual(t, cfg.workDir, staticEnvOpts.WorkDir)
	testutil.CheckDeepEqual(t, cfg.rpcPort, staticEnvOpts.RPCPort)
	testutil.CheckDeepEqual(t, cfg.httpPort, staticEnvOpts.HTTPPort)
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		description string
		input       interface{}
		expected    []string
	}{
		{
			description: "static env opts, all defined",
			input: StaticEnvOpts{
				DefaultRepo: util.StringPtr("gcr.io/foo"),
				RPCPort:     8080,
				HTTPPort:    8081,
				WorkDir:     "./foo",
			},
			expected: []string{
				"DEFAULT_REPO=gcr.io/foo",
				"RPC_PORT=8080",
				"HTTP_PORT=8081",
				"WORK_DIR=./foo",
			},
		},
		{
			description: "static env opts, some missing",
			input: StaticEnvOpts{
				RPCPort:  8080,
				HTTPPort: 8081,
				WorkDir:  "./foo",
			},
			expected: []string{
				"RPC_PORT=8080",
				"HTTP_PORT=8081",
				"WORK_DIR=./foo",
			},
		},
		{
			description: "build env opts",
			input: BuildEnvOpts{
				Image:        "foo",
				PushImage:    true,
				ImageRepo:    "gcr.io/foo",
				ImageTag:     "latest",
				BuildContext: "./foo",
			},
			expected: []string{
				"IMAGE=foo",
				"PUSH_IMAGE=true",
				"IMAGE_REPO=gcr.io/foo",
				"IMAGE_TAG=latest",
				"BUILD_CONTEXT=./foo",
			},
		},
		{
			description: "sync env opts, all defined",
			input: SyncEnvOpts{
				Image:                "foo",
				FilesAddedOrModified: util.StringPtr("./foo/1,./foo/2"),
				FilesDeleted:         util.StringPtr("./foo/3,./foo/4"),
				KubeContext:          "minikube",
				Namespaces:           "np1,np2,np3",
				BuildContext:         "./foo",
			},
			expected: []string{
				"IMAGE=foo",
				"FILES_ADDED_OR_MODIFIED=./foo/1,./foo/2",
				"FILES_DELETED=./foo/3,./foo/4",
				"KUBE_CONTEXT=minikube",
				"NAMESPACES=np1,np2,np3",
				"BUILD_CONTEXT=./foo",
			},
		},
		{
			description: "sync env opts, some missing",
			input: SyncEnvOpts{
				Image:        "foo",
				KubeContext:  "minikube",
				Namespaces:   "np1,np2,np3",
				BuildContext: "./foo",
			},
			expected: []string{
				"IMAGE=foo",
				"KUBE_CONTEXT=minikube",
				"NAMESPACES=np1,np2,np3",
				"BUILD_CONTEXT=./foo",
			},
		},
		{
			description: "deploy env opts",
			input: DeployEnvOpts{
				RunID:       "00000000-0000-0000-0000-​000000000000",
				KubeContext: "minikube",
				Namespaces:  "np1,np2,np3",
			},
			expected: []string{
				"RUN_ID=00000000-0000-0000-0000-​000000000000",
				"KUBE_CONTEXT=minikube",
				"NAMESPACES=np1,np2,np3",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual := getEnv(test.input)
			t.CheckElementsMatch(test.expected, actual)
		})
	}
}

type mockCfg struct {
	defaultRepo *string
	workDir     string
	rpcPort     int
	httpPort    int
}

func (m mockCfg) DefaultRepo() *string  { return m.defaultRepo }
func (m mockCfg) GetWorkingDir() string { return m.workDir }
func (m mockCfg) RPCPort() int          { return m.rpcPort }
func (m mockCfg) RPCHTTPPort() int      { return m.httpPort }
