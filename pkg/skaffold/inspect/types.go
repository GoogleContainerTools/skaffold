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

package inspect

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// Options holds flag values for the various `skaffold inspect` commands
type Options struct {
	// Filename is the `skaffold.yaml` file path
	Filename string
	// OutFormat is the output format. One of: json
	OutFormat string
	// Modules is the module filter for specific commands
	Modules []string

	ProfilesOptions
	BuildEnvOptions
}

// ProfilesOptions holds flag values for various `skaffold inspect profiles` commands
type ProfilesOptions struct {
	// BuildEnv is the build-env filter for command output. One of: local, googleCloudBuild, cluster.
	BuildEnv BuildEnv
}

// BuildEnvOptions holds flag values for various `skaffold inspect build-env` commands
type BuildEnvOptions struct {
	// Profiles is the slice of profile names to activate.
	Profiles []string
	// Profile is a target profile to create or edit
	Profile string
	// ProjectID is the GCP project ID
	ProjectID string
	// DiskSizeGb is the disk size of the VM that runs the build
	DiskSizeGb int64
	// MachineType is the type of VM that runs the build
	MachineType string
	// Timeout is the build timeout (in seconds)
	Timeout string
	// Concurrency is the number of artifacts to build concurrently. 0 means "no-limit"
	Concurrency int
	PullSecretPath string
	PullSecretName string
	PullSecretMountPath string
	Namespace string
	DockerConfigPath string
	DockerConfigSecretName string
	ServiceAccount string
	RunAsUser int64
	RandomPullSecret bool
	RandomDockerConigSecret bool
}

type BuildEnv string

var (
	ConfigSetFunc = parser.GetConfigSet
	BuildEnvs     = struct {
		Unspecified      BuildEnv
		Local            BuildEnv
		GoogleCloudBuild BuildEnv
		Cluster          BuildEnv
	}{
		Local: "local", GoogleCloudBuild: "googleCloudBuild", Cluster: "cluster",
	}
)

func GetBuildEnv(t *latestV1.BuildType) BuildEnv {
	switch {
	case t.Cluster != nil:
		return BuildEnvs.Cluster
	case t.GoogleCloudBuild != nil:
		return BuildEnvs.GoogleCloudBuild
	default:
		return BuildEnvs.Local
	}
}
