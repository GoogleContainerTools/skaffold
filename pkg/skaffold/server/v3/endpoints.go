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

package v3

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	event "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v3"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

var (
	// For Testing
	resetStateOnBuild  = event.ResetStateOnBuild
	resetStateOnDeploy = event.ResetStateOnDeploy
)

func (s *Server) GetState(context.Context, *empty.Empty) (*proto.State, error) {
	return event.GetState()
}

func (s *Server) Events(_ *empty.Empty, stream proto.SkaffoldV3Service_EventsServer) error {
	fmt.Println("v3 events")
	return event.ForEachEvent(stream.Send)
}

func (s *Server) ApplicationLogs(_ *empty.Empty, stream proto.SkaffoldV3Service_ApplicationLogsServer) error {
	return event.ForEachApplicationLog(stream.Send)
}

func (s *Server) Handle(ctx context.Context, e *proto.Event) (*empty.Empty, error) {
	return &empty.Empty{}, event.Handle(e)
}

func (s *Server) Execute(ctx context.Context, request *proto.UserIntentRequest) (*empty.Empty, error) {
	intent := request.GetIntent()
	if intent.GetBuild() {
		resetStateOnBuild()
		go func() {
			s.BuildIntentCallback()
		}()
	}

	if intent.GetDeploy() {
		resetStateOnDeploy()
		go func() {
			s.DeployIntentCallback()
		}()
	}

	if intent.GetSync() {
		go func() {
			s.SyncIntentCallback()
		}()
	}

	return &empty.Empty{}, nil
}

func (s *Server) AutoBuild(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger(constants.Build, request, event.UpdateStateAutoBuildTrigger, event.ResetStateOnBuild, s.AutoBuildCallback)
}

func (s *Server) AutoDeploy(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger(constants.Deploy, request, event.UpdateStateAutoDeployTrigger, event.ResetStateOnDeploy, s.AutoDeployCallback)
}

func (s *Server) AutoSync(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger(constants.Sync, request, event.UpdateStateAutoSyncTrigger, func() {}, s.AutoSyncCallback)
}

func executeAutoTrigger(triggerName constants.Phase, request *proto.TriggerRequest, updateTriggerStateFunc func(bool), resetPhaseStateFunc func(), serverCallback func(bool)) (res *empty.Empty, err error) {
	res = &empty.Empty{}

	trigger := request.GetState().GetEnabled()
	update, err := event.AutoTriggerDiff(triggerName, trigger)
	if err != nil {
		return
	}
	if !update {
		err = status.Errorf(codes.AlreadyExists, "auto %v is already set to %t", triggerName, trigger)
		return
	}
	// update trigger state
	updateTriggerStateFunc(trigger)
	if trigger {
		// reset phase state only when auto trigger is being set to true
		resetPhaseStateFunc()
	}
	go func() {
		serverCallback(trigger)
	}()
	return
}
