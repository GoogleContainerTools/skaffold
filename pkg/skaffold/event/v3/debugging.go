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
	proto "github.com/GoogleContainerTools/skaffold/proto/v3"
)

// DebuggingContainerStarted notifies that a debuggable container has appeared.
func DebuggingContainerStarted(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	debuggingContainerEvent := &proto.DebuggingContainerStartedEvent{
		Status:        Started,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		Artifact:      artifact,
		Runtime:       runtime,
		WorkingDir:    workingDir,
		DebugPorts:    debugPorts,
	}
	WrapInMainAndHandle(artifact, debuggingContainerEvent, DebuggingContainerStartedEvent)
}

// DebuggingContainerTerminated notifies that a debuggable container has disappeared.
func DebuggingContainerTerminated(podName, containerName, namespace, artifact, runtime, workingDir string, debugPorts map[string]uint32) {
	debuggingContainerEvent := &proto.DebuggingContainerTerminatedEvent{
		Status:        Terminated,
		PodName:       podName,
		ContainerName: containerName,
		Namespace:     namespace,
		Artifact:      artifact,
		Runtime:       runtime,
		WorkingDir:    workingDir,
		DebugPorts:    debugPorts,
	}
	WrapInMainAndHandle(artifact, debuggingContainerEvent, DebuggingContainerTerminatedEvent)
}
