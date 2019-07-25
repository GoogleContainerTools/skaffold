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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

func (s *server) GetState(context.Context, *empty.Empty) (*proto.State, error) {
	return event.GetState()
}

func (s *server) EventLog(stream proto.SkaffoldService_EventLogServer) error {
	return event.ForEachEvent(stream.Send)
}

func (s *server) Events(stream proto.SkaffoldService_EventsServer) error {
	return event.ForEachEvent(stream.Send)
}

func (s *server) Handle(ctx context.Context, e *proto.Event) (*empty.Empty, error) {
	event.Handle(e)
	return &empty.Empty{}, nil
}

func (s *server) Execute(ctx context.Context, intent *proto.UserIntentRequest) (*empty.Empty, error) {
	if intent.GetIntent().GetBuild() {
		go func() {
			s.buildIntentCallback()
		}()
	}

	if intent.GetIntent().GetDeploy() {
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
