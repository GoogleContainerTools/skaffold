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

package docker

import (
	"context"
	"io"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	dockerport "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/docker/port"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type fakeDockerDaemon struct {
	docker.LocalDaemon
	PulledImages []string
	ImgsInDaemon map[string]string
}

func (fd *fakeDockerDaemon) NetworkCreate(ctx context.Context, name string, labels map[string]string) error {
	return nil
}

func (fd *fakeDockerDaemon) Pull(ctx context.Context, out io.Writer, ref string, platform v1.Platform) error {
	fd.PulledImages = append(fd.PulledImages, ref)
	return nil
}

func (fd *fakeDockerDaemon) ImageID(ctx context.Context, ref string) (string, error) {
	img := fd.ImgsInDaemon[ref]
	return img, nil
}

func getActionCfg(aName string, containers []latest.VerifyContainer) latest.Action {
	return latest.Action{
		Name: aName,
		Config: latest.ActionConfig{
			IsFailFast: util.Ptr(true),
			Timeout:    util.Ptr(0),
		},
		ExecutionModeConfig: latest.ActionExecutionModeConfig{
			VerifyExecutionModeType: latest.VerifyExecutionModeType{
				LocalExecutionMode: &latest.LocalVerifier{},
			},
		},
		Containers: containers,
	}
}
func TestExecEnv_PrepareActions(t *testing.T) {
	tests := []struct {
		description      string
		actionToExec     string
		expectedTasks    []string
		shouldFail       bool
		errMsg           string
		availableAcsCfgs []latest.Action
	}{
		{
			description:  "fail with action not found",
			actionToExec: "not-created-action",
			shouldFail:   true,
			errMsg:       "action not-created-action not found for local execution mode",
			availableAcsCfgs: []latest.Action{
				getActionCfg("action1", []latest.VerifyContainer{}),
			},
		},
		{
			description:   "prepare action with two tasks",
			actionToExec:  "action1",
			expectedTasks: []string{"container1", "container2"},
			availableAcsCfgs: []latest.Action{
				getActionCfg("action1", []latest.VerifyContainer{
					{Name: "container1", Image: "gcr.io/k8s-skaffold/mock:latest"},
					{Name: "container2", Image: "gcr.io/k8s-skaffold/mock:latest"},
				}),
				getActionCfg("action2", []latest.VerifyContainer{
					{Name: "container3", Image: "gcr.io/k8s-skaffold/mock:latest"},
					{Name: "container4", Image: "gcr.io/k8s-skaffold/mock:latest"},
				}),
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.TODO()

			// Mock Docker client.
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return &fakeDockerDaemon{
					LocalDaemon: docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil),
				}, nil
			})

			// Mock new tasks creation to track created actions.
			var createdTasks []string
			t.Override(&NewTask, func(c latest.VerifyContainer, client docker.LocalDaemon, pM *dockerport.PortManager, resources []*latest.PortForwardResource, artifact graph.Artifact, timeout int, execEnv *ExecEnv) Task {
				createdTasks = append(createdTasks, c.Name)
				return newTask(c, client, pM, resources, artifact, timeout, execEnv)
			})

			execEnv, err := NewExecEnv(ctx, nil, &label.DefaultLabeller{}, nil, "", nil, test.availableAcsCfgs)
			t.CheckNoError(err)

			acs, err := execEnv.PrepareActions(ctx, nil, nil, nil, []string{test.actionToExec})

			if test.shouldFail {
				t.CheckErrorContains(test.errMsg, err)
			} else {
				t.CheckNoError(err)
				t.CheckDeepEqual(1, len(acs))
				a := acs[0]
				t.CheckDeepEqual(test.actionToExec, a.Name())
				t.CheckDeepEqual(test.expectedTasks, createdTasks)
			}
		})
	}
}

func TestExecEnv_PullImages(t *testing.T) {
	tests := []struct {
		description        string
		actionToExec       string
		availableAcsCfgs   []latest.Action
		expectedPulledImgs []string
		builtImgs          []graph.Artifact
		imagesInDaemon     map[string]string
	}{
		{
			description:  "prepare action with two images to pull",
			actionToExec: "action1",
			availableAcsCfgs: []latest.Action{
				getActionCfg("action1", []latest.VerifyContainer{
					{Name: "container1", Image: "gcr.io/k8s-skaffold/mock1:latest"},
					{Name: "container2", Image: "gcr.io/k8s-skaffold/mock2:latest"},
				}),
			},
			expectedPulledImgs: []string{"gcr.io/k8s-skaffold/mock1:latest", "gcr.io/k8s-skaffold/mock2:latest"},
		},
		{
			description:  "prepare action with one image to pull and other already built and present in daemon",
			actionToExec: "action1",
			availableAcsCfgs: []latest.Action{
				getActionCfg("action1", []latest.VerifyContainer{
					{Name: "container1", Image: "mock1"},
					{Name: "container2", Image: "mock2:latest"},
				}),
			},
			expectedPulledImgs: []string{"mock2:latest"},
			builtImgs: []graph.Artifact{
				{
					ImageName: "mock1",
					Tag:       "mock1:latest",
				},
			},
			imagesInDaemon: map[string]string{
				"mock1:latest": "id1234",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.TODO()

			// Mock Docker client.
			fDockerDaemon := &fakeDockerDaemon{
				LocalDaemon:  docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil),
				ImgsInDaemon: test.imagesInDaemon,
			}
			t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
				return fDockerDaemon, nil
			})

			execEnv, err := NewExecEnv(ctx, nil, &label.DefaultLabeller{}, nil, "", nil, test.availableAcsCfgs)
			t.CheckNoError(err)

			_, err = execEnv.PrepareActions(ctx, nil, test.builtImgs, nil, []string{test.actionToExec})
			t.CheckNoError(err)

			t.CheckDeepEqual(test.expectedPulledImgs, fDockerDaemon.PulledImages)
		})
	}
}
