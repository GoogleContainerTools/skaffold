package v2

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
)

func CloudRunServiceReady(r, url, revision string) {
	handler.handleCloudRunReady(&proto.CloudRunReadyEvent{
		Id:            r,
		TaskId:        fmt.Sprintf("%s-%d", constants.Deploy, handler.iteration),
		Resource:      r,
		Url:           url,
		ReadyRevision: revision,
	})
}

func (ev *eventHandler) handleCloudRunReady(e *proto.CloudRunReadyEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_CloudRunReadyEvent{
			CloudRunReadyEvent: e,
		},
	})
}
