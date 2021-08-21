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

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func TestResourceStatusCheckEventUpdated(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	ResourceStatusCheckEventUpdated("ns:pod/foo", proto.ActionableErr{
		ErrCode: 509,
		Message: "image pull error",
	})
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == InProgress })
}

func TestResourceStatusCheckEventSucceeded(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventSucceeded("ns:pod/foo")
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Succeeded })
}

func TestResourceStatusCheckEventFailed(t *testing.T) {
	defer func() { handler = newHandler() }()

	handler = newHandler()
	handler.state = emptyState(mockCfg([]latestV1.Pipeline{{}}, "test"))

	wait(t, func() bool { return handler.getState().StatusCheckState.Status == NotStarted })
	resourceStatusCheckEventFailed("ns:pod/foo", proto.ActionableErr{
		ErrCode: 309,
		Message: "one or more deployments failed",
	})
	wait(t, func() bool { return handler.getState().StatusCheckState.Resources["ns:pod/foo"] == Failed })
}