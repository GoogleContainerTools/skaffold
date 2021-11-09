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
	"fmt"

	pbuf "github.com/golang/protobuf/proto"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func GetState() (*proto.State, error) {
	state := handler.getState()
	return state, nil
}

func (ev *eventHandler) getState() *proto.State {
	ev.stateLock.Lock()
	// Deep copy
	state := pbuf.Clone(ev.state).(*proto.State)
	ev.stateLock.Unlock()

	return state
}

func (ev *eventHandler) setState(state *proto.State) {
	ev.stateLock.Lock()
	ev.state = state
	ev.stateLock.Unlock()
}

// InitializeState instantiates the global state of the skaffold runner, as well as the event log.
func InitializeState(cfg Config) {
	handler.cfg = cfg
	handler.setState(emptyState(cfg))
}

func emptyState(cfg Config) *proto.State {
	builds := map[string]string{}
	for _, p := range cfg.GetPipelines() {
		for _, a := range p.Build.Artifacts {
			builds[a.ImageName] = NotStarted
		}
	}
	metadata := initializeMetadata(cfg.GetPipelines(), cfg.GetKubeContext(), cfg.GetRunID())
	return emptyStateWithArtifacts(builds, metadata, cfg.AutoBuild(), cfg.AutoDeploy(), cfg.AutoSync())
}

func emptyStateWithArtifacts(builds map[string]string, metadata *proto.Metadata, autoBuild, autoDeploy, autoSync bool) *proto.State {
	return &proto.State{
		BuildState: &proto.BuildState{
			Artifacts:   builds,
			AutoTrigger: autoBuild,
			StatusCode:  proto.StatusCode_OK,
		},
		TestState: &proto.TestState{
			Status:     NotStarted,
			StatusCode: proto.StatusCode_OK,
		},
		RenderState: &proto.RenderState{
			Status:     NotStarted,
			StatusCode: proto.StatusCode_OK,
		},
		DeployState: &proto.DeployState{
			Status:      NotStarted,
			AutoTrigger: autoDeploy,
			StatusCode:  proto.StatusCode_OK,
		},
		StatusCheckState: emptyStatusCheckState(),
		ForwardedPorts:   make(map[int32]*proto.PortForwardEvent),
		FileSyncState: &proto.FileSyncState{
			Status:      NotStarted,
			AutoTrigger: autoSync,
		},
		Metadata: metadata,
	}
}

func emptyStatusCheckState() *proto.StatusCheckState {
	return &proto.StatusCheckState{
		Status:     NotStarted,
		Resources:  map[string]string{},
		StatusCode: proto.StatusCode_OK,
	}
}

// ResetStateOnBuild resets the build, test, deploy and sync state
func ResetStateOnBuild() {
	builds := map[string]string{}
	for k := range handler.getState().BuildState.Artifacts {
		builds[k] = NotStarted
	}
	autoBuild, autoDeploy, autoSync := handler.getState().BuildState.AutoTrigger, handler.getState().DeployState.AutoTrigger, handler.getState().FileSyncState.AutoTrigger
	newState := emptyStateWithArtifacts(builds, handler.getState().Metadata, autoBuild, autoDeploy, autoSync)
	handler.setState(newState)
}

// ResetStateOnTest resets the test, deploy, sync and status check state
func ResetStateOnTest() {
	newState := handler.getState()
	newState.TestState.Status = NotStarted
	handler.setState(newState)
}

// ResetStateOnDeploy resets the deploy, sync and status check state
func ResetStateOnDeploy() {
	newState := handler.getState()
	newState.DeployState.Status = NotStarted
	newState.DeployState.StatusCode = proto.StatusCode_OK
	newState.StatusCheckState = emptyStatusCheckState()
	newState.ForwardedPorts = map[int32]*proto.PortForwardEvent{}
	newState.DebuggingContainers = nil
	handler.setState(newState)
}

func UpdateStateAutoBuildTrigger(t bool) {
	newState := handler.getState()
	newState.BuildState.AutoTrigger = t
	handler.setState(newState)
}

func UpdateStateAutoDeployTrigger(t bool) {
	newState := handler.getState()
	newState.DeployState.AutoTrigger = t
	handler.setState(newState)
}

func UpdateStateAutoSyncTrigger(t bool) {
	newState := handler.getState()
	newState.FileSyncState.AutoTrigger = t
	handler.setState(newState)
}

func AutoTriggerDiff(phase constants.Phase, val bool) (bool, error) {
	switch phase {
	case constants.Build:
		return val != handler.getState().BuildState.AutoTrigger, nil
	case constants.Sync:
		return val != handler.getState().FileSyncState.AutoTrigger, nil
	case constants.Deploy:
		return val != handler.getState().DeployState.AutoTrigger, nil
	default:
		return false, fmt.Errorf("unknown Phase %v not found in handler state", phase)
	}
}
