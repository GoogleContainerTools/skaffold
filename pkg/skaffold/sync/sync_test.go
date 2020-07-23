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
package sync

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	registryv1 "github.com/google/go-containerregistry/pkg/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewSyncItem(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		description  string
		artifact     *latest.Artifact
		dependencies map[string][]string
		labels       map[string]string
		evt          filemon.Events
		builds       []build.Artifact
		shouldErr    bool
		expected     *Item
		workingDir   string
	}{
		// manual sync cases
		{
			description: "manual: match copy",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{{Src: "*.html", Dest: "."}},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					"index.html": {"index.html"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "manual: no tag for image",
			artifact: &latest.Artifact{
				ImageName: "notbuildyet",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{{Src: "*.html", Dest: "."}},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			shouldErr: true,
		},
		{
			description: "manual: multiple sync patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "*.js", Dest: "."},
						{Src: "*.html", Dest: "."},
						{Src: "*.json", Dest: "."},
					},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
				Deleted:  []string{filepath.Join("node", "package.json")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "server.js"):  {"server.js"},
					filepath.Join("node", "index.html"): {"index.html"},
				},
				Delete: map[string][]string{
					filepath.Join("node", "package.json"): {"package.json"},
				},
			},
		},
		{
			description: "manual: recursive glob patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "src/**/*.js", Dest: "src/", Strip: "src/"},
					},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Modified: []string{filepath.Join("node", "src/app/server/server.js")},
			},
			workingDir: "/",
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "src/app/server/server.js"): {"/src/app/server/server.js"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "manual: sync all",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "*", Dest: "."},
					},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
				Deleted:  []string{filepath.Join("node", "package.json")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "server.js"):  {"server.js"},
					filepath.Join("node", "index.html"): {"index.html"},
				},
				Delete: map[string][]string{
					filepath.Join("node", "package.json"): {"package.json"},
				},
			},
		},
		{
			description: "manual: not copy syncable",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "*.html", Dest: "."},
					},
				},
				Workspace: ".",
			},
			evt: filemon.Events{
				Added:   []string{"main.go"},
				Deleted: []string{"index.html"},
			},
			builds: []build.Artifact{
				{
					Tag: "placeholder",
				},
			},
		},
		{
			description: "manual: not delete syncable",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "*.html", Dest: "/static"},
					},
				},
				Workspace: ".",
			},
			evt: filemon.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
			builds: []build.Artifact{
				{
					Tag: "placeholder",
				},
			},
		},
		{
			description: "manual: err bad pattern",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "[*.html", Dest: "*"},
					},
				},
				Workspace: ".",
			},
			evt: filemon.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"some/other/file"},
			},
			shouldErr: true,
		},
		{
			description: "manual: no change no sync",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "*.html", Dest: "*"},
					},
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
			description: "manual: slashes in glob pattern",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "**/**/*.js", Dest: "."},
					},
				},
				Workspace: ".",
			},
			workingDir: "/some",
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{filepath.Join("dir1", "dir2", "node.js")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("dir1", "dir2", "node.js"): {"/some/dir1/dir2/node.js"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "manual: sync subtrees",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "dir1/**/*.js", Dest: ".", Strip: "dir1/"},
					},
				},
				Workspace: ".",
			},
			workingDir: "/some/dir",
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{filepath.Join("dir1", "dir2/node.js")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("dir1", "dir2/node.js"): {"/some/dir/dir2/node.js"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "manual: multiple matches",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "dir1/**/*.js", Dest: ".", Strip: "dir1/"},
						{Src: "dir1/**/**/*.js", Dest: "."},
					},
				},
				Workspace: ".",
			},
			workingDir: "/some",
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{filepath.Join("dir1", "dir2", "node.js")},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("dir1", "dir2", "node.js"): {"/some/dir2/node.js", "/some/dir1/dir2/node.js"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "manual: stars work with absolute paths",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Manual: []*latest.SyncRule{
						{Src: "dir1a/**/*.js", Dest: "/tstar", Strip: "dir1a/"},
						{Src: "dir1b/**/*.js", Dest: "/dstar"},
					},
				},
				Workspace: ".",
			},
			workingDir: "/some/dir",
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{
					filepath.Join("dir1a", "dir2", "dir3", "node.js"),
					filepath.Join("dir1b", "dir1", "node.js"),
				},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("dir1a", "dir2", "dir3", "node.js"): {"/tstar/dir2/dir3/node.js"},
					filepath.Join("dir1b", "dir1", "node.js"):         {"/dstar/dir1b/dir1/node.js"},
				},
				Delete: map[string][]string{},
			},
		},

		// auto-sync cases
		{
			description: "infer: match copy",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"*.html"},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					"index.html": {"/index.html"},
				},
			},
		},
		{
			description: "infer: not auto-syncable",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"*.html"},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			dependencies: map[string][]string{},
		},
		{
			description: "infer: file not specified for syncing",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"*.js"},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}},
		},
		{
			description: "infer: no tag for image",
			artifact: &latest.Artifact{
				ImageName: "notbuildyet",
				Sync: &latest.Sync{
					Infer: []string{"*.html"},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}},
			shouldErr:    true,
		},
		{
			description: "infer: multiple sync patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"*.js", "*.html", "*.json"},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}, "server.js": {"/server.js"}},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "server.js"):  {"/server.js"},
					filepath.Join("node", "index.html"): {"/index.html"},
				},
			},
		},
		{
			description: "infer: recursive glob patterns",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"src/**/*.js"},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Modified: []string{filepath.Join("node", "src", "app", "server", "server.js")},
			},
			dependencies: map[string][]string{filepath.Join("src", "app", "server", "server.js"): {"/dest/server.js"}},
			workingDir:   "/",
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "src", "app", "server", "server.js"): {"/dest/server.js"},
				},
			},
		},
		{
			description: "infer: sync all",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"*"},
				},
				Workspace: "node",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added:    []string{filepath.Join("node", "index.html")},
				Modified: []string{filepath.Join("node", "server.js")},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}, "server.js": {"/server.js"}},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("node", "server.js"):  {"/server.js"},
					filepath.Join("node", "index.html"): {"/index.html"},
				},
			},
		},
		{
			description: "infer: delete not syncable",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Infer: []string{"*"},
				},
				Workspace: ".",
			},
			evt: filemon.Events{
				Added:   []string{"index.html"},
				Deleted: []string{"server.html"},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}, "server.html": {"/server.html"}},
			builds: []build.Artifact{
				{
					Tag: "placeholder",
				},
			},
		},
		{
			description: "infer: err bad pattern",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Infer: []string{"[*.html"},
				},
				Workspace: ".",
			},
			evt: filemon.Events{
				Added: []string{"index.html"},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}},
			shouldErr:    true,
		},
		{
			description: "infer: no change no sync",
			artifact: &latest.Artifact{
				Sync: &latest.Sync{
					Infer: []string{"*.html"},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			dependencies: map[string][]string{"index.html": {"/index.html"}},
		},
		{
			description: "infer: slashes in glob pattern",
			artifact: &latest.Artifact{
				ImageName: "test",
				Sync: &latest.Sync{
					Infer: []string{"**/**/*.js"},
				},
				Workspace: ".",
			},
			workingDir: "/some",
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{filepath.Join("dir1", "dir2", "node.js")},
			},
			dependencies: map[string][]string{filepath.Join("dir1", "dir2", "node.js"): {"/some/node.js"}},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					filepath.Join("dir1", "dir2", "node.js"): {"/some/node.js"},
				},
			},
		},

		// Buildpacks
		{
			description: "auto with buildpacks",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{},
				},
				ImageName: "test",
				Sync: &latest.Sync{
					Auto: &latest.Auto{},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"file.go"},
			},
			labels: map[string]string{
				"io.buildpacks.build.metadata": `{
					"bom":[{
						"metadata":{
							"devmode.sync": [
								{"src":"*.go","dest":"/some"}
							]
						}
					}]
				}`,
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					"file.go": {"/some/file.go"},
				},
				Delete: map[string][]string{},
			},
		},
		{
			description: "unknown change with buildpacks",
			artifact: &latest.Artifact{
				ArtifactType: latest.ArtifactType{
					BuildpackArtifact: &latest.BuildpackArtifact{},
				},
				ImageName: "test",
				Sync: &latest.Sync{
					Auto: &latest.Auto{},
				},
				Workspace: ".",
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			evt: filemon.Events{
				Added: []string{"unknown"},
			},
			labels: map[string]string{
				"io.buildpacks.build.metadata": `{
					"bom":[{
						"metadata":{
							"devmode.sync": [
								{"src":"*.go","dest":"/some"}
							]
						}
					}]
				}`,
			},
			expected: nil,
		},

		// Auto with Jib
		{
			description: "auto with jib",
			artifact: &latest.Artifact{
				ImageName: "test",
				Workspace: ".",
				Sync: &latest.Sync{
					Auto: &latest.Auto{},
				},
				ArtifactType: latest.ArtifactType{
					JibArtifact: &latest.JibArtifact{},
				},
			},
			evt: filemon.Events{
				Added: []string{"this actually doesn't matter"},
			},
			builds: []build.Artifact{
				{
					ImageName: "test",
					Tag:       "test:123",
				},
			},
			expected: &Item{
				Image: "test:123",
				Copy: map[string][]string{
					"file.class": {"/some/file.class"},
				},
				Delete: nil,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&WorkingDir, func(string, map[string]bool) (string, error) { return test.workingDir, nil })
			t.Override(&SyncMap, func(*latest.Artifact, map[string]bool) (map[string][]string, error) { return test.dependencies, nil })
			t.Override(&Labels, func(string, map[string]bool) (map[string]string, error) { return test.labels, nil })
			t.Override(&jib.GetSyncDiff, func(context.Context, string, *latest.JibArtifact, filemon.Events) (map[string][]string, map[string][]string, error) {
				return map[string][]string{"file.class": {"/some/file.class"}}, nil, nil
			})

			actual, err := NewItem(ctx, test.artifact, test.evt, test.builds, nil)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		description string
		syncRules   []*latest.SyncRule
		files       []string
		context     string
		workingDir  string
		expected    syncMap
		shouldErr   bool
	}{
		{
			description: "nil sync patterns doesn't sync",
			expected:    map[string][]string{},
		},
		{
			description: "copy nested file to correct destination",
			files:       []string{filepath.Join("static", "index.html"), filepath.Join("static", "test.html")},
			syncRules: []*latest.SyncRule{
				{Src: filepath.Join("static", "*.html"), Dest: "/html", Strip: "static/"},
			},
			expected: map[string][]string{
				filepath.Join("static", "index.html"): {"/html/index.html"},
				filepath.Join("static", "test.html"):  {"/html/test.html"},
			},
		},
		{
			description: "double-star matches depth zero",
			files:       []string{"index.html"},
			syncRules: []*latest.SyncRule{
				{Src: filepath.Join("**", "*.html"), Dest: "/html"},
			},
			expected: map[string][]string{
				"index.html": {"/html/index.html"},
			},
		},
		{
			description: "file not in . copies to correct destination",
			files:       []string{filepath.Join("node", "server.js")},
			context:     "node",
			syncRules: []*latest.SyncRule{
				{Src: "*.js", Dest: "/"},
			},
			expected: map[string][]string{
				filepath.Join("node", "server.js"): {"/server.js"},
			},
		},
		{
			description: "file change not relative to context throws error",
			files:       []string{filepath.Join("node", "server.js"), filepath.Join("/", "something", "test.js")},
			context:     "node",
			syncRules: []*latest.SyncRule{
				{Src: "*.js", Dest: "/"},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actual, err := intersect(test.context, test.workingDir, test.syncRules, test.files)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
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

func fakeCmd(ctx context.Context, p v1.Pod, c v1.Container, files syncMap) *exec.Cmd {
	var args []string

	for src, dsts := range files {
		for _, dst := range dsts {
			args = append(args, src, dst)
		}
	}

	return exec.CommandContext(ctx, "copy", args...)
}

var pod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "podname",
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "skaffold",
		},
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

var nonRunningPod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name: "podname",
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "skaffold",
		},
	},
	Status: v1.PodStatus{
		Phase: v1.PodPending,
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
	tests := []struct {
		description string
		image       string
		files       syncMap
		pod         *v1.Pod
		cmdFn       func(context.Context, v1.Pod, v1.Container, syncMap) *exec.Cmd
		cmdErr      error
		clientErr   error
		expected    []string
		shouldErr   bool
	}{
		{
			description: "no error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       syncMap{"test.go": {"/test.go"}},
			pod:         pod,
			cmdFn:       fakeCmd,
			expected:    []string{"copy test.go /test.go"},
		},
		{
			description: "cmd error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       syncMap{"test.go": {"/test.go"}},
			pod:         pod,
			cmdFn:       fakeCmd,
			cmdErr:      fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "client error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       syncMap{"test.go": {"/test.go"}},
			pod:         pod,
			cmdFn:       fakeCmd,
			clientErr:   fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "no copy",
			image:       "gcr.io/different-pod:123",
			files:       syncMap{"test.go": {"/test.go"}},
			pod:         pod,
			cmdFn:       fakeCmd,
			shouldErr:   true,
		},
		{
			description: "Skip sync when pod is not running",
			image:       "gcr.io/k8s-skaffold:123",
			files:       syncMap{"test.go": {"/test.go"}},
			pod:         nonRunningPod,
			cmdFn:       fakeCmd,
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			cmdRecord := &TestCmdRecorder{err: test.cmdErr}

			t.Override(&util.DefaultExecCommand, cmdRecord)
			t.Override(&pkgkubernetes.Client, func() (kubernetes.Interface, error) {
				return fake.NewSimpleClientset(test.pod), test.clientErr
			})

			err := Perform(context.Background(), test.image, test.files, test.cmdFn, []string{""})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, cmdRecord.cmds)
		})
	}
}

func TestSyncMap(t *testing.T) {
	tests := []struct {
		description  string
		artifactType latest.ArtifactType
		files        map[string]string
		shouldErr    bool
		expectedMap  map[string][]string
	}{
		{
			description: "docker - supported",
			artifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: "Dockerfile",
				},
			},
			files: map[string]string{
				"Dockerfile": "FROM alpine\nCOPY *.go /app/",
				"main.go":    "",
			},
			expectedMap: map[string][]string{"main.go": {"/app/main.go"}},
		},
		{
			description: "kaniko - supported",
			artifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{
					DockerfilePath: "Dockerfile",
				},
			},
			files: map[string]string{
				"Dockerfile": "FROM alpine\nCOPY *.go /app/",
				"main.go":    "",
			},
			expectedMap: map[string][]string{"main.go": {"/app/main.go"}},
		},
		{
			description: "custom - supported",
			artifactType: latest.ArtifactType{
				CustomArtifact: &latest.CustomArtifact{
					Dependencies: &latest.CustomDependencies{
						Dockerfile: &latest.DockerfileDependency{
							Path: "Dockerfile",
						},
					},
				},
			},
			files: map[string]string{
				"Dockerfile": "FROM alpine\nCOPY *.go /app/",
				"main.go":    "",
			},
			expectedMap: map[string][]string{"main.go": {"/app/main.go"}},
		},
		{
			description: "custom, no dockerfile - not supported",
			artifactType: latest.ArtifactType{
				CustomArtifact: &latest.CustomArtifact{},
			},
			shouldErr: true,
		},
		{
			description:  "not supported",
			artifactType: latest.ArtifactType{},
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			imageFetcher := fakeImageFetcher{}
			t.Override(&docker.RetrieveImage, imageFetcher.fetch)
			t.NewTempDir().WriteFiles(test.files).Chdir()

			syncMap, err := SyncMap(&latest.Artifact{ArtifactType: test.artifactType}, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expectedMap, syncMap)
		})
	}
}

type fakeImageFetcher struct{}

func (f *fakeImageFetcher) fetch(image string, _ map[string]bool) (*registryv1.ConfigFile, error) {
	return &registryv1.ConfigFile{}, nil
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		description string
		artifact    *latest.Artifact
		shouldInit  bool
		initErrors  bool
	}{
		{
			description: "sync off",
			artifact:    &latest.Artifact{},
			shouldInit:  false,
		},
		{
			description: "sync on, auto off",
			artifact:    &latest.Artifact{Sync: &latest.Sync{}},
			shouldInit:  false,
		},
		{
			description: "sync on, auto on, non-jib",
			artifact:    &latest.Artifact{Sync: &latest.Sync{Auto: &latest.Auto{}}},
			shouldInit:  false,
		},
		{
			description: "sync on, auto on, jib",
			artifact:    &latest.Artifact{ArtifactType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}}, Sync: &latest.Sync{Auto: &latest.Auto{}}},
			shouldInit:  true,
			initErrors:  false,
		},
		{
			description: "sync on, auto on, jib, init fails",
			artifact:    &latest.Artifact{ArtifactType: latest.ArtifactType{JibArtifact: &latest.JibArtifact{}}, Sync: &latest.Sync{Auto: &latest.Auto{}}},
			shouldInit:  true,
			initErrors:  true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			isCalled := false
			t.Override(&jib.InitSync, func(ctx context.Context, workspace string, a *latest.JibArtifact) error {
				isCalled = true
				if test.initErrors {
					return errors.New("intentional test failure")
				}
				return nil
			})

			artifacts := []*latest.Artifact{test.artifact}
			err := Init(ctx, artifacts)
			t.CheckDeepEqual(test.shouldInit, isCalled)
			t.CheckError(test.initErrors, err)
		})
	}
}
