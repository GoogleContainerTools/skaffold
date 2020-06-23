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

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func (s *server) GetState(context.Context, *empty.Empty) (*proto.State, error) {
	return event.GetState()
}

func (s *server) EventLog(stream proto.SkaffoldService_EventLogServer) error {
	return event.ForEachEvent(stream.Send)
}

func (s *server) Events(_ *empty.Empty, stream proto.SkaffoldService_EventsServer) error {
	return event.ForEachEvent(stream.Send)
}

func (s *server) Handle(ctx context.Context, e *proto.Event) (*empty.Empty, error) {
	event.Handle(e)
	return &empty.Empty{}, nil
}

func (s *server) Execute(ctx context.Context, intent *proto.UserIntentRequest) (*empty.Empty, error) {
	if intent.GetIntent().GetBuild() {
		event.ResetStateOnBuild()
		go func() {
			s.buildIntentCallback()
		}()
	}

	if intent.GetIntent().GetDeploy() {
		event.ResetStateOnDeploy()
		go func() {
			s.deployIntentCallback()
		}()
	}

	if intent.GetIntent().GetSync() {
		go func() {
			s.syncIntentCallback()
		}()
	}

	return &empty.Empty{}, nil
}

func (s *server) AutoBuild(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger("build", request, event.UpdateStateAutoBuildTrigger, event.ResetStateOnBuild, s.autoBuildCallback)
}

func (s *server) AutoDeploy(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger("deploy", request, event.UpdateStateAutoDeployTrigger, event.ResetStateOnDeploy, s.autoDeployCallback)
}

func (s *server) AutoSync(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	return executeAutoTrigger("sync", request, event.UpdateStateAutoSyncTrigger, func() {}, s.autoSyncCallback)
}

func executeAutoTrigger(triggerName string, request *proto.TriggerRequest, updateTriggerStateFunc func(bool), resetPhaseStateFunc func(), serverCallback func(bool)) (res *empty.Empty, err error) {
	res = &empty.Empty{}
	v, ok := request.GetState().GetVal().(*proto.TriggerState_Enabled)
	if !ok {
		err = status.Error(codes.InvalidArgument, "missing required boolean parameter 'enabled'")
		return
	}
	trigger := v.Enabled
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
