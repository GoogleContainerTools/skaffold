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
	res = &empty.Empty{}
	autoBuild := request.GetState().Enabled
	updateAutoBuild, err := event.AutoTriggerDiff("build", autoBuild)
	if err != nil {
		return
	}
	if !updateAutoBuild {
		return
	}
	event.UpdateStateAutoBuildTrigger(autoBuild)
	if autoBuild {
		// reset state only when autoBuild is being set to true
		event.ResetStateOnBuild()
	}
	go func() {
		s.autoBuildCallback(autoBuild)
	}()
	return
}

func (s *server) AutoDeploy(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	res = &empty.Empty{}
	autoDeploy := request.GetState().Enabled
	updateAutoDeploy, err := event.AutoTriggerDiff("deploy", autoDeploy)
	if err != nil {
		return
	}
	if !updateAutoDeploy {
		return
	}

	event.UpdateStateAutoDeployTrigger(autoDeploy)
	if autoDeploy {
		// reset state only when autoDeploy is being set to true
		event.ResetStateOnDeploy()
	}
	go func() {
		s.autoDeployCallback(autoDeploy)
	}()
	return
}

func (s *server) AutoSync(ctx context.Context, request *proto.TriggerRequest) (res *empty.Empty, err error) {
	res = &empty.Empty{}
	autoSync := request.GetState().Enabled
	updateAutoSync, err := event.AutoTriggerDiff("sync", autoSync)
	if err != nil {
		return
	}
	if !updateAutoSync {
		return
	}
	event.UpdateStateAutoSyncTrigger(autoSync)
	go func() {
		s.autoSyncCallback(autoSync)
	}()
	return
}
