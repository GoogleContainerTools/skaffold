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

package config

import (
	"strings"
	"time"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// WaitForDeletions configures the wait for pending deletions.
type WaitForDeletions struct {
	Max     time.Duration
	Delay   time.Duration
	Enabled bool
}

// SkaffoldOptions are options that are set by command line arguments not included
// in the config file itself
type SkaffoldOptions struct {
	CacheFile         string
	ConfigurationFile string
	Command           string
	CustomTag         string
	DigestSource      string
	EventLogFile      string
	GlobalConfig      string
	KubeConfig        string
	KubeContext       string
	LastLogFile       string
	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles
	MinikubeProfile     string
	Namespace           string
	RenderOutput        string
	RepoCacheDir        string
	Trigger             string
	User                string
	TransformRulesFile  string
	VerifyDockerNetwork string

	ConfigurationFilter []string
	CustomLabels        []string
	HydratedManifests   []string
	InsecureRegistries  []string
	Profiles            []string
	TargetImages        []string

	Apply                 bool
	AutoBuild             bool
	AutoCreateConfig      bool
	AutoDeploy            bool
	AutoSync              bool
	AssumeYes             bool
	CacheArtifacts        bool
	ContainerDebugging    bool
	Cleanup               bool
	DetectMinikube        bool
	DryRun                bool
	EnableRPC             bool
	Force                 bool
	ForceLoadImages       bool
	IterativeStatusCheck  bool
	Notification          bool
	NoPrune               bool
	NoPruneChildren       bool
	ProfileAutoActivation bool
	PropagateProfiles     bool
	RenderOnly            bool
	SkipTests             bool
	SkipRender            bool
	SkipConfigDefaults    bool
	Tail                  bool
	WaitForConnection     bool
	// Add Skaffold-specific labels including runID, deployer labels, etc.
	// `CustomLabels` are still applied if this is false. Must only be used in
	// commands which don't deploy (e.g. `skaffold render`) since the runID
	// label isn't available.
	AddSkaffoldLabels bool
	MakePathsAbsolute *bool
	StatusCheck       BoolOrUndefined
	PortForward       PortForwardOptions
	DefaultRepo       StringOrUndefined
	PushImages        BoolOrUndefined
	Platforms         []string
	Muted             Muted
	BuildConcurrency  int
	WatchPollInterval int
	RPCPort           IntOrUndefined
	RPCHTTPPort       IntOrUndefined

	MultiLevelRepo   *bool
	SyncRemoteCache  SyncRemoteCacheOption
	WaitForDeletions WaitForDeletions
}

type RunMode string

var RunModes = struct {
	Build    RunMode
	Dev      RunMode
	Debug    RunMode
	Run      RunMode
	Deploy   RunMode
	Render   RunMode
	Delete   RunMode
	Diagnose RunMode
}{
	Build:    "build",
	Dev:      "dev",
	Debug:    "debug",
	Run:      "run",
	Deploy:   "deploy",
	Render:   "render",
	Delete:   "delete",
	Diagnose: "diagnose",
}

// Prune returns true iff the user did NOT specify the --no-prune flag,
// and the user did NOT specify the --cache-artifacts flag.
func (opts *SkaffoldOptions) Prune() bool {
	return !opts.NoPrune && !opts.CacheArtifacts
}

func (opts *SkaffoldOptions) Mode() RunMode {
	return RunMode(opts.Command)
}

func (opts *SkaffoldOptions) IsTargetImage(artifact *latestV1.Artifact) bool {
	if len(opts.TargetImages) == 0 {
		return true
	}

	for _, targetImage := range opts.TargetImages {
		if strings.Contains(artifact.ImageName, targetImage) {
			return true
		}
	}

	return false
}
