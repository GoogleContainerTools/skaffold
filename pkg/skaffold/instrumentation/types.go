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

package instrumentation

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// skaffoldMeter describes the data used to determine operational metrics.
type skaffoldMeter struct {
	// ConfigCount is the number of parsed skaffold configurations in the current session.
	ConfigCount int

	// ExitCode Exit code returned by Skaffold at the end of execution.
	ExitCode int

	// BuildArtifacts Number of artifacts built in the current execution as defined in skaffold.yaml.
	BuildArtifacts int

	// Command Command that is used to execute skaffold `dev, build, render, run, etc.`
	// without any command-line arguments.
	Command string

	// Version Version of Skaffold being used "v1.18.0, v1.19.1, etc.".
	Version string

	// OS OS running Skaffold e.g. Windows, Linux, Darwin, etc.
	OS string

	// Arch Architecture running Skaffold e.g. amd64, arm64, etc.
	Arch string

	// PlatformType Where Skaffold is building artifacts (local, cluster, Google Cloud Build, or a combination of them).
	PlatformType string

	// User indicates the client invoking skaffold. Is one of allowedUser i.e. vsc, intellij, gcloud
	User string

	// Deployers All the deployers used in the Skaffold execution.
	Deployers []string

	// EnumFlags Enum values for flags passed into Skaffold that have a pre-defined list of
	// valid values e.g. `'–cache-artifacts=false', '–mute-logs=["build", "deploy"]'`.
	EnumFlags map[string]string

	// Builders Enum values for all the builders used to build the artifacts built.
	Builders map[string]int

	// BuildWithPlatforms Enum values for all the builders with target platform constraints.
	BuildWithPlatforms map[string]int

	// BuildDependencies Enum values for all the builders using build dependencies.
	BuildDependencies map[string]int

	// MultiHelmReleasesCount is the number of releases if helm deployer is present.
	HelmReleasesCount int

	// SyncType Sync type used in the build configuration: infer, auto, and/or manual.
	SyncType map[string]bool

	// Hooks Enum values for all the configured lifecycle hooks.
	Hooks map[HookPhase]int

	// DevIterations Error results of the various dev iterations and the
	// reasons they were triggered. The triggers can be one of sync, build, or deploy.
	DevIterations []devIteration

	// StartTime Start time of the Skaffold program, used to track how long Skaffold took to finish executing.
	StartTime time.Time

	// Duration Time Skaffold took to finish executing in milliseconds.
	Duration time.Duration

	// ErrorCode Skaffold reports [error codes](/docs/references/api/grpc/#statuscode)
	// and these are monitored in order to determine the most frequent errors.
	ErrorCode proto.StatusCode

	// ClusterType reports if user cluster is a GKE cluster or not.
	ClusterType string

	// ResolvedBuildTargetPlatforms represents the set of resolved build target platforms for each pipeline
	ResolvedBuildTargetPlatforms []string

	// CliBuildTargetPlatforms represents the build target platforms specified via command line flag `--platform`
	CliBuildTargetPlatforms string

	// DeployNodePlatforms represents the set of kubernetes cluster node platforms
	DeployNodePlatforms string
}

// devIteration describes how an iteration and started and if an error happened.
type devIteration struct {
	// Intent is the cause of initiating the dev iteration (sync, build, deploy).
	Intent string

	// ErrorCode is the error that may have occurred during the (sync/build/deploy).
	ErrorCode proto.StatusCode
}

// creds contains the Google Cloud project ID.
type creds struct {
	// ProjectID is the ID of the Google Cloud project to upload metrics to.
	ProjectID string `json:"project_id"`
}

// errHandler prints errors to logrus.
type errHandler struct{}

func (h errHandler) Handle(err error) {
	log.Entry(context.TODO()).Debugf("Error with metrics: %v", err)
}

type HookPhase string

var HookPhases = struct {
	PreBuild   HookPhase
	PostBuild  HookPhase
	PreSync    HookPhase
	PostSync   HookPhase
	PreDeploy  HookPhase
	PostDeploy HookPhase
}{
	PreBuild:   "pre-build",
	PostBuild:  "post-build",
	PreSync:    "pre-sync",
	PostSync:   "post-sync",
	PreDeploy:  "pre-deploy",
	PostDeploy: "post-deploy",
}
