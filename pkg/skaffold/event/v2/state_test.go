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

package v2

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetState(t *testing.T) {
	ev := newHandler()
	ev.state = emptyState(mockCfg([]latest.Pipeline{{}}, "test"))

	ev.stateLock.Lock()
	ev.state.BuildState.Artifacts["img"] = Complete
	ev.stateLock.Unlock()

	state := ev.getState()

	testutil.CheckDeepEqual(t, Complete, state.BuildState.Artifacts["img"])
}

func TestResetStateOnBuild(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		RenderState: &proto.RenderState{Status: Complete},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState:    &proto.FileSyncState{Status: Succeeded},
	}

	ResetStateOnBuild()
	expected := &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": NotStarted,
			},
		},
		TestState:        &proto.TestState{Status: NotStarted},
		RenderState:      &proto.RenderState{Status: NotStarted},
		DeployState:      &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted, Resources: map[string]string{}},
		FileSyncState:    &proto.FileSyncState{Status: NotStarted},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestResetStateOnDeploy(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: Complete},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
	}
	ResetStateOnDeploy()
	expected := &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
		},
		DeployState: &proto.DeployState{Status: NotStarted},
		StatusCheckState: &proto.StatusCheckState{Status: NotStarted,
			Resources: map[string]string{},
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestEmptyStateCheckState(t *testing.T) {
	actual := emptyStatusCheckState()
	expected := &proto.StatusCheckState{Status: NotStarted,
		Resources: map[string]string{},
	}
	testutil.CheckDeepEqual(t, expected, actual, cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestUpdateStateAutoTriggers(t *testing.T) {
	defer func() { handler = newHandler() }()
	handler = newHandler()
	handler.state = &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: false,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: false},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState: &proto.FileSyncState{
			Status:      "Complete",
			AutoTrigger: false,
		},
	}
	UpdateStateAutoBuildTrigger(true)
	UpdateStateAutoDeployTrigger(true)
	UpdateStateAutoSyncTrigger(true)

	expected := &proto.State{
		BuildState: &proto.BuildState{
			Artifacts: map[string]string{
				"image1": Complete,
			},
			AutoTrigger: true,
		},
		DeployState: &proto.DeployState{Status: Complete, AutoTrigger: true},
		ForwardedPorts: map[int32]*proto.PortForwardEvent{
			2001: {
				LocalPort:  2000,
				PodName:    "test/pod",
				TargetPort: &targetPort,
			},
		},
		StatusCheckState: &proto.StatusCheckState{Status: Complete},
		FileSyncState: &proto.FileSyncState{
			Status:      "Complete",
			AutoTrigger: true,
		},
	}
	testutil.CheckDeepEqual(t, expected, handler.getState(), cmpopts.EquateEmpty(), protocmp.Transform())
}

func TestAutoTriggerDiff(t *testing.T) {
	tests := []struct {
		description  string
		phase        constants.Phase
		handlerState *proto.State
		val          bool
		expected     bool
	}{
		{
			description: "build needs update",
			phase:       constants.Build,
			val:         true,
			handlerState: &proto.State{
				BuildState: &proto.BuildState{
					AutoTrigger: false,
				},
			},
			expected: true,
		},
		{
			description: "deploy doesn't need update",
			phase:       constants.Deploy,
			val:         true,
			handlerState: &proto.State{
				BuildState: &proto.BuildState{
					AutoTrigger: false,
				},
				DeployState: &proto.DeployState{
					AutoTrigger: true,
				},
			},
			expected: false,
		},
		{
			description: "sync needs update",
			phase:       constants.Sync,
			val:         false,
			handlerState: &proto.State{
				FileSyncState: &proto.FileSyncState{
					AutoTrigger: true,
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Setup handler state
			handler.setState(test.handlerState)

			got, err := AutoTriggerDiff(test.phase, test.val)
			if err != nil {
				t.Fail()
			}

			t.CheckDeepEqual(test.expected, got)
		})
	}
}
