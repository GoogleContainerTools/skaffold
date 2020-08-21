/*
Copyright 2020 The Skaffold Authors

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
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKpt_Deploy(t *testing.T) {
	tests := []struct {
		description string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "nil",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k := NewKptDeployer(&runcontext.RunContext{}, nil)
			res, err := k.Deploy(context.Background(), nil, nil)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, res)
		})
	}
}

func TestKpt_Dependencies(t *testing.T) {
	tests := []struct {
		description    string
		cfg            *latest.KptDeploy
		createFiles    map[string]string
		kustomizations map[string]string
		expected       []string
		shouldErr      bool
	}{
		{
			description: "bad dir",
			cfg: &latest.KptDeploy{
				Dir: "invalid_path",
			},
			shouldErr: true,
		},
		{
			description: "empty dir and unspecified fnPath",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
		},
		{
			description: "dir",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			createFiles: map[string]string{
				"foo.yaml":  "",
				"README.md": "",
			},
			expected: []string{"foo.yaml"},
		},
		{
			description: "dir with subdirs and file path variants",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			createFiles: map[string]string{
				"food.yml":           "",
				"foo/bar.yaml":       "",
				"foo/bat//bad.yml":   "",
				"foo/bat\\README.md": "",
			},
			expected: []string{"foo/bar.yaml", "foo/bat/bad.yml", "food.yml"},
		},
		{
			description: "fnpath",
			cfg: &latest.KptDeploy{
				Dir: ".",
				Fn:  latest.KptFn{FnPath: "kpt-func.yaml"},
			},
			expected: []string{"kpt-func.yaml"},
		},
		{
			description: "fnpath and dir and kustomization",
			cfg: &latest.KptDeploy{
				Dir: ".",
				Fn:  latest.KptFn{FnPath: "kpt-func.yaml"},
			},
			createFiles: map[string]string{"foo.yml": ""},
			kustomizations: map[string]string{"kustomization.yaml": `configMapGenerator:
- files: [app1.properties]`},
			expected: []string{"app1.properties", "foo.yml", "kpt-func.yaml", "kustomization.yaml"},
		},
		{
			description: "dependencies that can only be detected as a kustomization",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			kustomizations: map[string]string{"kustomization.yaml": `configMapGenerator:
- files: [app1.properties]`},
			expected: []string{"app1.properties", "kustomization.yaml"},
		},
		{
			description: "kustomization.yml variant",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			kustomizations: map[string]string{"kustomization.yml": `configMapGenerator:
- files: [app1.properties]`},
			expected: []string{"app1.properties", "kustomization.yml"},
		},
		{
			description: "Kustomization variant",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			kustomizations: map[string]string{"Kustomization": `configMapGenerator:
- files: [app1.properties]`},
			expected: []string{"Kustomization", "app1.properties"},
		},
		{
			description: "incorrectly named kustomization",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			kustomizations: map[string]string{"customization": `configMapGenerator:
- files: [app1.properties]`},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().Chdir()

			tmpDir.WriteFiles(test.createFiles)
			tmpDir.WriteFiles(test.kustomizations)

			k := NewKptDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy: test.cfg,
						},
					},
				},
			}, nil)

			res, err := k.Dependencies()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, tmpDir.Paths(test.expected...), tmpDir.Paths(res...))
		})
	}
}

func TestKpt_Cleanup(t *testing.T) {
	tests := []struct {
		description string
		applyDir    string
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "invalid user specified applyDir",
			applyDir:    "invalid_path",
			shouldErr:   true,
		},
		{
			description: "valid user specified applyDir w/o template resource",
			applyDir:    "valid_path",
			commands:    testutil.CmdRunOutErr("kpt live destroy valid_path", "", errors.New("BUG")),
			shouldErr:   true,
		},
		{
			description: "valid user specified applyDir w/ template resource (emulated)",
			applyDir:    "valid_path",
			commands:    testutil.CmdRunOut("kpt live destroy valid_path", ""),
		},
		{
			description: "unspecified applyDir",
			commands: testutil.
				CmdRunOut("kpt live init .kpt-hydrated", "").
				AndRunOut("kpt live destroy .kpt-hydrated", ""),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.NewTempDir().Chdir()

			if test.applyDir == "valid_path" {
				os.Mkdir(test.applyDir, 0755)
			}

			k := NewKptDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy: &latest.KptDeploy{
								ApplyDir: test.applyDir,
							},
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			}, nil)

			err := k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKpt_Render(t *testing.T) {
	output1 := `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
`

	output2 := `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`

	output3 := `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`

	output4 := `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`

	tests := []struct {
		description string
		builds      []build.Artifact
		labels      map[string]string
		cfg         *latest.KptDeploy
		commands    util.Command
		expected    string
		shouldErr   bool
	}{
		{
			description: "no fnPath or image specified",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
			},
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			commands: testutil.CmdRunOut("kpt fn run . --dry-run", output1),
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
`,
		},
		{
			description: "fnPath specified",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			cfg: &latest.KptDeploy{
				Dir: "test",
				Fn:  latest.KptFn{FnPath: "kpt-func.yaml"},
			},
			commands: testutil.CmdRunOut("kpt fn run test --dry-run --fn-path kpt-func.yaml", output2),
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "image specified",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			cfg: &latest.KptDeploy{
				Dir: "test",
				Fn:  latest.KptFn{Image: "gcr.io/example.com/my-fn:v1.0.0 -- foo=bar"},
			},
			commands: testutil.CmdRunOut("kpt fn run test --dry-run --image gcr.io/example.com/my-fn:v1.0.0 -- foo=bar", output3),
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "multiple resources outputted",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			cfg: &latest.KptDeploy{
				Dir: "test",
				Fn:  latest.KptFn{FnPath: "kpt-func.yaml"},
			},
			commands: testutil.CmdRunOut("kpt fn run test --dry-run --fn-path kpt-func.yaml", output4),
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "user labels",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
			},
			labels: map[string]string{"user/label": "test"},
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			commands: testutil.CmdRunOut("kpt fn run . --dry-run", output1),
			expected: `apiVersion: v1
kind: Pod
metadata:
  labels:
    user/label: test
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
`,
		},
		{
			description: "empty output from pipeline",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
			},
			labels: map[string]string{"user/label": "test"},
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			commands: testutil.CmdRunOut("kpt fn run . --dry-run", ``),
			expected: "\n",
		},
		{
			description: "kpt fn run fails",
			cfg: &latest.KptDeploy{
				Dir: ".",
			},
			commands:  testutil.CmdRunOutErr("kpt fn run . --dry-run", "invalid pipeline", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "both fnPath and image specified",
			cfg: &latest.KptDeploy{
				Dir: "test",
				Fn: latest.KptFn{
					FnPath: "kpt-func.yaml",
					Image:  "gcr.io/example.com/my-fn:v1.0.0 -- foo=bar"},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			k := NewKptDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			}, test.labels)

			var b bytes.Buffer
			err := k.Render(context.Background(), &b, test.builds, true, "")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, b.String())
		})
	}
}

func TestKpt_GetApplyDir(t *testing.T) {
	tests := []struct {
		description string
		applyDir    string
		expected    string
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "specified an invalid applyDir",
			applyDir:    "invalid_path",
			shouldErr:   true,
		},
		{
			description: "specified a valid applyDir",
			applyDir:    "valid_path",
			expected:    "valid_path",
		},
		{
			description: "unspecified applyDir",
			expected:    ".kpt-hydrated",
			commands:    testutil.CmdRunOut("kpt live init .kpt-hydrated", ""),
		},
		{
			description: "existing template resource in .kpt-hydrated",
			expected:    ".kpt-hydrated",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			tmpDir := t.NewTempDir().Chdir()

			if test.applyDir == test.expected {
				os.Mkdir(test.applyDir, 0755)
			}

			if test.description == "existing template resource in .kpt-hydrated" {
				tmpDir.Touch(".kpt-hydrated/inventory-template.yaml")
			}

			k := NewKptDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KptDeploy: &latest.KptDeploy{
								ApplyDir: test.applyDir,
							},
						},
					},
				},
			}, nil)

			applyDir, err := k.getApplyDir(context.Background())

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, applyDir)
		})
	}
}

func TestKpt_KptCommandArgs(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		commands    []string
		flags       []string
		globalFlags []string
		expected    []string
	}{
		{
			description: "empty",
		},
		{
			description: "all inputs have len >0",
			dir:         "test",
			commands:    []string{"live", "apply"},
			flags:       []string{"--fn-path", "kpt-func.yaml"},
			globalFlags: []string{"-h"},
			expected:    strings.Split("live apply test --fn-path kpt-func.yaml -h", " "),
		},
		{
			description: "empty dir",
			commands:    []string{"live", "apply"},
			flags:       []string{"--fn-path", "kpt-func.yaml"},
			globalFlags: []string{"-h"},
			expected:    strings.Split("live apply --fn-path kpt-func.yaml -h", " "),
		},
		{
			description: "empty commands",
			dir:         "test",
			flags:       []string{"--fn-path", "kpt-func.yaml"},
			globalFlags: []string{"-h"},
			expected:    strings.Split("test --fn-path kpt-func.yaml -h", " "),
		},
		{
			description: "empty flags",
			dir:         "test",
			commands:    []string{"live", "apply"},
			globalFlags: []string{"-h"},
			expected:    strings.Split("live apply test -h", " "),
		},
		{
			description: "empty globalFlags",
			dir:         "test",
			commands:    []string{"live", "apply"},
			flags:       []string{"--fn-path", "kpt-func.yaml"},
			expected:    strings.Split("live apply test --fn-path kpt-func.yaml", " "),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			res := kptCommandArgs(test.dir, test.commands, test.flags, test.globalFlags)
			t.CheckDeepEqual(test.expected, res)
		})
	}
}
