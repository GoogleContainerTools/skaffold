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
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

const (
	Cache = "Cache"
	Build = "Build"
	Push  = "Push"
)

var artifactIDs = map[string]int{}

func AssignArtifactIDs(artifacts []*latestV1.Artifact) {
	for i, a := range artifacts {
		artifactIDs[a.ImageName] = i
	}
}

func CacheCheckInProgress(id int, artifact string) {
	buildSubtaskEvent(id, artifact, Cache, Started, nil)
}

func CacheCheckMiss(id int, artifact string) {
	buildSubtaskEvent(id, artifact, Cache, Failed, nil)
}

func CacheCheckHit(id int, artifact string) {
	buildSubtaskEvent(id, artifact, Cache, Succeeded, nil)
}

func BuildInProgress(id int, artifact string) {
	buildSubtaskEvent(id, artifact, Build, Started, nil)
}

func BuildFailed(id int, artifact string, err error) {
  buildSubtaskEvent(id, artifact, Build, Failed, err)
}

func BuildSucceeded(id int, artifact string) {
	buildSubtaskEvent(id, artifact, Build, Succeeded, nil)
}

func buildSubtaskEvent(id int, artifact, step, status string, err error) {
	var aErr *proto.ActionableErr
	if err != nil {
		aErr = sErrors.ActionableErrV2(handler.cfg, constants.Build, err)
	}
	handler.handleBuildSubtaskEvent(&proto.BuildSubtaskEvent{
		Id:            strconv.Itoa(id),
		TaskId:        fmt.Sprintf("%s-%d", constants.Build, handler.iteration),
		Artifact:      artifact,
		Step:          step,
		Status:        status,
		ActionableErr: aErr,
	})
}

func (ev *eventHandler) handleBuildSubtaskEvent(e *proto.BuildSubtaskEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_BuildSubtaskEvent{
			BuildSubtaskEvent: e,
		},
	})
}
