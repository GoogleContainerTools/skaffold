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
	protoV3 "github.com/GoogleContainerTools/skaffold/proto/v3"
)

// DebuggingContainerStarted notifies that a debuggable container has appeared.
func DebuggingContainerStarted(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	debuggingContainerEvent := &protoV3.DebuggingContainerStartedEvent{
		Status:        Started,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		Artifact:      artifact,
		Runtime:       runtime,
		WorkingDir:    workingDir,
		DebugPorts:    debugPorts,
	}

	handler.stateLock.Lock()
	handler.state.DebuggingContainers = append(handler.state.DebuggingContainers, &protoV3.DebuggingContainerState{
		Id:            debuggingContainerEvent.Id,
		TaskId:        debuggingContainerEvent.TaskId,
		Status:        debuggingContainerEvent.Status,
		PodName:       debuggingContainerEvent.PodName,
		ContainerName: debuggingContainerEvent.ContainerName,
		Namespace:     debuggingContainerEvent.Namespace,
		Artifact:      debuggingContainerEvent.Artifact,
		Runtime:       debuggingContainerEvent.Runtime,
		WorkingDir:    debuggingContainerEvent.WorkingDir,
		DebugPorts:    debuggingContainerEvent.DebugPorts,
	})
	handler.stateLock.Unlock()

	handler.handle(debuggingContainerEvent, DebuggingContainerStartedEvent)
}

// DebuggingContainerTerminated notifies that a debuggable container has disappeared.
func DebuggingContainerTerminated(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	debuggingContainerEvent := &protoV3.DebuggingContainerTerminatedEvent{
		Status:        Terminated,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		Artifact:      artifact,
		Runtime:       runtime,
		WorkingDir:    workingDir,
		DebugPorts:    debugPorts,
	}

	handler.stateLock.Lock()
	n := 0
	for _, x := range handler.state.DebuggingContainers {
		if x.Namespace != debuggingContainerEvent.Namespace || x.PodName != debuggingContainerEvent.PodName || x.ContainerName != debuggingContainerEvent.ContainerName {
			handler.state.DebuggingContainers[n] = x
			n++
		}
	}
	handler.state.DebuggingContainers = handler.state.DebuggingContainers[:n]
	handler.stateLock.Unlock()

	handler.handle(debuggingContainerEvent, DebuggingContainerTerminatedEvent)
}