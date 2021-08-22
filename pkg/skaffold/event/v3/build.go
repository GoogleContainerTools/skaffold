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
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

const (
	Cache = "Cache"
	Build = "Build"
)

func CacheCheckInProgress(artifact string) {
	buildEvent := &proto.BuildStartedEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Cache,
		Status:        InProgress,
		ActionableErr: nil,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildStartedEvent)
}

func CacheCheckMiss(artifact string) {
	buildEvent := &proto.BuildFailedEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Cache,
		Status:        Failed,
		ActionableErr: nil,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildFailedEvent)
}

func CacheCheckHit(artifact string) {
	buildEvent := &proto.BuildSucceededEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Cache,
		Status:        Succeeded,
		ActionableErr: nil,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildSucceededEvent)
}

func BuildInProgress(artifact string) {
	fmt.Println("Build Started event")
	buildEvent := &proto.BuildStartedEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Build,
		Status:        InProgress,
		ActionableErr: nil,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildStartedEvent)
}

func BuildFailed(artifact string, err error) {
	var aErr *proto.ActionableErr
	if err != nil {
		aErr = sErrors.ActionableErrV3(handler.cfg, constants.Build, err)
	}
	buildEvent := &proto.BuildFailedEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Build,
		Status:        Failed,
		ActionableErr: aErr,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildFailedEvent)
}

func BuildSucceeded(artifact string) {
	buildEvent := &proto.BuildSucceededEvent{
		Id:            artifact,
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Build,
		Status:        Succeeded,
		ActionableErr: nil,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildSucceededEvent)
}

func BuildCanceled(artifact string, err error) {
	var aErr *proto.ActionableErr
	if err != nil {
		aErr = sErrors.ActionableErrV3(handler.cfg, constants.Build, err)
	}
	buildEvent := &proto.BuildCancelledEvent{
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          Build,
		Status:        Canceled,
		ActionableErr: aErr,
	}
	WrapInMainAndHandle(artifact, buildEvent, BuildCancelledEvent)
}
