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

import (
	"strings"
)

const (
	// DebugConfig is the name of the podspec annotation that records debugging configuration information.
	// The annotation should be a JSON-encoded map of container-name to a `ContainerDebugConfiguration` object.
	DebugConfig = "debug.cloud.google.com/config"

	// DebugProbesAnnotation is the name of the podspec annotation that disables rewriting of probe timeouts.
	// The annotation value should be `skip`.
	DebugProbeTimeouts = "debug.cloud.google.com/probe/timeouts"
)

// ContainerDebugConfiguration captures debugging information for a specific container.
// This structure is serialized out and included in the pod metadata.
type ContainerDebugConfiguration struct {
	// Artifact is the corresponding artifact's image name used in the skaffold.yaml
	Artifact string `json:"artifact,omitempty"`
	// Runtime represents the underlying language runtime (`go`, `jvm`, `nodejs`, `python`, `netcore`)
	Runtime string `json:"runtime,omitempty"`
	// WorkingDir is the working directory in the image configuration; may be empty
	WorkingDir string `json:"workingDir,omitempty"`
	// Ports is the list of debugging ports, keyed by protocol type
	Ports map[string]uint32 `json:"ports,omitempty"`
}

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

// Runtime specifies the target language runtime for this artifact that is used to configure debug support
type Runtime string

var Runtimes = struct {
	Go      Runtime
	NodeJS  Runtime
	JVM     Runtime
	Python  Runtime
	NetCore Runtime
	Unknown Runtime
}{
	Go:      "go",
	NodeJS:  "nodejs",
	JVM:     "jvm",
	Python:  "python",
	NetCore: "netcore",
	Unknown: "unknown",
}

func ToRuntime(r string) Runtime {
	switch strings.ToLower(r) {
	case "go", "golang":
		return Runtimes.Go
	case "nodejs", "node", "npm":
		return Runtimes.NodeJS
	case "jvm", "java":
		return Runtimes.JVM
	case "python":
		return Runtimes.Python
	case "netcore", ".net", "dotnet":
		return Runtimes.NetCore
	default:
		return Runtimes.Unknown
	}
}
