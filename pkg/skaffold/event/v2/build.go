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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	proto "github.com/GoogleContainerTools/skaffold/v2/proto/v2"
)

const (
	Cache = "Cache"
	Build = "Build"
)

func CacheCheckInProgress(artifact, platforms string) {
	buildSubtaskEvent(artifact, platforms, Cache, InProgress, nil)
}

func CacheCheckMiss(artifact, platforms string) {
	buildSubtaskEvent(artifact, platforms, Cache, Failed, nil)
}

func CacheCheckHit(artifact, platforms string) {
	buildSubtaskEvent(artifact, platforms, Cache, Succeeded, nil)
}

func BuildInProgress(artifact, platforms string) {
	buildSubtaskEvent(artifact, platforms, Build, InProgress, nil)
}

func BuildFailed(artifact, platforms string, err error) {
	buildSubtaskEvent(artifact, platforms, Build, Failed, err)
}

func BuildSucceeded(artifact, platforms string) {
	buildSubtaskEvent(artifact, platforms, Build, Succeeded, nil)
}

func BuildCanceled(artifact, platforms string, err error) {
	buildSubtaskEvent(artifact, platforms, Build, Canceled, err)
}

func buildSubtaskEvent(artifact, platforms, step, status string, err error) {
	var aErr *proto.ActionableErr
	if err != nil {
		aErr = sErrors.ActionableErrV2(handler.cfg, constants.Build, err)
		handler.sendErrorMessage(constants.Build, artifact, err)
	}
	handler.handleBuildSubtaskEvent(&proto.BuildSubtaskEvent{
		Id:              artifact,
		TaskId:          fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:        artifact,
		TargetPlatforms: platforms,
		HostPlatform:    platform.Host.String(),
		Step:            step,
		Status:          status,
		ActionableErr:   aErr,
	})
}

func (ev *eventHandler) handleBuildSubtaskEvent(e *proto.BuildSubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_BuildSubtaskEvent{
			BuildSubtaskEvent: e,
		},
	})
}
