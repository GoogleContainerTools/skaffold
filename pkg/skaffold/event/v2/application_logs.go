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

import proto "github.com/GoogleContainerTools/skaffold/proto/v2"

func ApplicationLog(podName, containerName, prefix, message, formattedMessage string) {
	handler.handleApplicationLogEvent(&proto.ApplicationLogEvent{
		ContainerName:        containerName,
		PodName:              podName,
		Prefix:               prefix,
		Message:              message,
		RichFormattedMessage: formattedMessage,
	})
}

func (ev *eventHandler) handleApplicationLogEvent(e *proto.ApplicationLogEvent) {
	ev.handle(&proto.Event{
		EventType: &proto.Event_ApplicationLogEvent{
			ApplicationLogEvent: e,
		},
	})
}
