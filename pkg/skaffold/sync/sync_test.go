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
package sync

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewSyncItem(t *testing.T) {
	var tests = []struct {
		description string
		artifact    *latest.Artifact
		evt         watch.Events
		builds      []build.Artifact
		shouldErr   bool
		expected    *Item
	}{
		{
			description: "match copy",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added: []string{"index.html"},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					"index.html": "index.html",
				},
				Delete: map[string]string{},
			},
		},
		{
			description: "no tag for image",
			artifact: &latest.Artifact{
				ImageName: "notbuildyet",
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added: []string{"index.html"},
			},
			shouldErr: true,
		},
		{
			description: "multiple sync patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*.js":   ".",
					"*.html": ".",
					"*.json": ".",
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
				Deleted:  []string{filepath.Join("node", "package.json")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					filepath.Join("node", "server.js"):  "server.js",
					filepath.Join("node", "index.html"): "index.html",
				},
				Delete: map[string]string{
					filepath.Join("node", "package.json"): "package.json",
				},
			},
		},
		{
			description: "recursive glob patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"src/**/*.js": "src/",
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Modified: []string{filepath.Join("node", "src/app/server/server.js")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					filepath.Join("node", "src/app/server/server.js"): "src/app/server/server.js",
				},
				Delete: map[string]string{},
			},
		},
		{
			description: "sync all",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"*": ".",
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
				Deleted:  []string{filepath.Join("node", "package.json")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					filepath.Join("node", "server.js"):  "server.js",
					filepath.Join("node", "index.html"): "index.html",
				},
				Delete: map[string]string{
					filepath.Join("node", "package.json"): "package.json",
				},
			},
		},
		{
			description: "not copy syncable",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": ".",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"main.go"},
				Deleted: []string{"index.html"},
			},
		},
		{
			description: "not delete syncable",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": "/static",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
		},
		{
			description: "err bad pattern",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"[*.html": "*",
				},
				Workspace: ".",
			},
			evt: watch.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
			shouldErr: true,
		},
		{
			description: "no change no sync",
			artifact: &latest.Artifact{
				Sync: map[string]string{
					"*.html": "*",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
		},
		{
			description: "slashes in glob pattern",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: map[string]string{
					"**/**/*.js": ".",
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: watch.Events{
				Added: []string{filepath.Join("dir1", "dir2/node.js")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string]string{
					filepath.Join("dir1", "dir2/node.js"): "dir1/dir2/node.js",
				},
				Delete: map[string]string{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := NewItem(test.artifact, test.evt, test.builds)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}

}

func TestIntersect(t *testing.T) {
	var tests = []struct {
		description  string
		syncPatterns map[string]string
		files        []string
		context      string
		expected     map[string]string
		shouldErr    bool
	}{
		{
			description: "nil sync patterns doesn't sync",
			expected:    map[string]string{},
		},
		{
			description: "copy nested file to correct destination",
			files:       []string{filepath.Join("static", "index.html"), filepath.Join("static", "test.html")},
			syncPatterns: map[string]string{
				filepath.Join("static", "*.html"): "/html",
			},
			expected: map[string]string{
				filepath.Join("static", "index.html"): "/html/index.html",
				filepath.Join("static", "test.html"):  "/html/test.html",
			},
		},
		{
			description: "file not in . copies to correct destination",
			files:       []string{filepath.Join("node", "server.js")},
			context:     "node",
			syncPatterns: map[string]string{
				"*.js": "/",
			},
			expected: map[string]string{
				filepath.Join("node", "server.js"): "/server.js",
			},
		},
		{
			description: "file change not relative to context throws error",
			files:       []string{filepath.Join("node", "server.js"), filepath.Join("/", "something", "test.js")},
			context:     "node",
			syncPatterns: map[string]string{
				"*.js": "/",
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual, err := intersect(test.context, test.syncPatterns, test.files)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}

type TestCmdRecorder struct {
	cmds []string
	err  error
}

func (t *TestCmdRecorder) RunCmd(cmd *exec.Cmd) error {
	if t.err != nil {
		return t.err
	}
	t.cmds = append(t.cmds, strings.Join(cmd.Args, " "))
	return nil
}

func (t *TestCmdRecorder) RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	return nil, t.RunCmd(cmd)
}

func fakeCmd(ctx context.Context, p v1.Pod, c v1.Container, src, dst string) []*exec.Cmd {
	return []*exec.Cmd{exec.CommandContext(ctx, "copy", src, dst)}
}

var pod = &v1.Pod{
	ObjectMeta: meta_v1.ObjectMeta{
		Name:   "podname",
		Labels: constants.Labels.DefaultLabels,
	},
	Status: v1.PodStatus{
		Phase: v1.PodRunning,
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "container_name",
				Image: "gcr.io/k8s-skaffold:123",
			},
		},
	},
}

func TestPerform(t *testing.T) {
	var tests = []struct {
		description string
		image       string
		files       map[string]string
		cmdFn       func(context.Context, v1.Pod, v1.Container, string, string) []*exec.Cmd
		cmdErr      error
		clientErr   error
		expected    []string
		shouldErr   bool
	}{
		{
			description: "no error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			expected:    []string{"copy test.go /test.go"},
		},
		{
			description: "cmd error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			cmdErr:      fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "client error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			clientErr:   fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "no copy",
			image:       "gcr.io/different-pod:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cmdRecord := &TestCmdRecorder{err: test.cmdErr}
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = cmdRecord

			defer func(c func() (kubernetes.Interface, error)) { pkgkubernetes.Client = c }(pkgkubernetes.GetClientset)
			pkgkubernetes.Client = func() (kubernetes.Interface, error) {
				return fake.NewSimpleClientset(pod), test.clientErr
			}

			util.DefaultExecCommand = cmdRecord

			err := Perform(context.Background(), test.image, test.files, test.cmdFn, []string{""})

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cmdRecord.cmds)
		})
	}
}
