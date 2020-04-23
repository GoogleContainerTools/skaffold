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

package validator

type ErrorCode int

const (
	NoError              = 0
	Unknown              = 1
	UnknownUnSchedulable = 2

	// Container errors
	ImagePullErr            = 51
	ContainerCreating       = 52
	RunContainerError       = 53
	ContainerTerminated     = 54
	ContainerWaitingUnknown = 55
	ContainerRestarting     = 56

	// K8 infra errors
	NodeMemoryPressure     = 100
	NodeDiskPressure       = 101
	NodeNetworkUnavailable = 102
	NodePIDPressure        = 103
	NodeUnschedulable      = 104
	NodeUnreachable        = 105
	NodeNotReady           = 106
)
