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

package types

// ContainerAdapter provides a surface to abstract away the underlying container
// representation which can be operated on by the debug transformers.
type ContainerAdapter interface {
	GetContainer() *ExecutableContainer
	Apply()
}

// ExecutableContainer holds shared fields between container representations.
// These fields are mutated by the debug transformers, and are eventually
// propagated back to the underlying container representation in the adapter.
type ExecutableContainer struct {
	Name    string
	Command []string
	Args    []string
	Env     ContainerEnv
	Ports   []ContainerPort
}

// adapted from github.com/kubernetes/api/core/v1/types.go
type ContainerPort struct {
	Name          string
	HostPort      int32
	ContainerPort int32
	Protocol      string
	HostIP        string
}

type ContainerEnv struct {
	Order []string
	Env   map[string]string
}
