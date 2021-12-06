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

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

// WaitForDeletions configures the wait for pending deletions.
type WaitForDeletions struct {
	Max     time.Duration
	Delay   time.Duration
	Enabled bool
}

// SkaffoldOptions are options that are set by command line arguments not included in the config file itself
type SkaffoldOptions struct {
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
	AddSkaffoldLabels  bool
	ConfigurationFile  string
	HydrationDir       string
	InventoryNamespace string
	InventoryID        string
	InventoryName      string
	GlobalConfig       string
	EventLogFile       string
	RenderOutput       string
	User               string
	CustomTag          string
	Namespace          string
	CacheFile          string
	Trigger            string
	KubeContext        string
	KubeConfig         string
	LastLogFile        string
	DigestSource       string
	Command            string
	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles
	MinikubeProfile     string
	RepoCacheDir        string
	CustomLabels        []string
	TargetImages        []string
	Profiles            []string
	InsecureRegistries  []string
	ConfigurationFilter []string
	HydratedManifests   []string
	BuildConcurrency    int
	WatchPollInterval   int

	DefaultRepo       StringOrUndefined
	MakePathsAbsolute *bool
	Muted             Muted
	PortForward       PortForwardOptions
	PushImages        BoolOrUndefined
	RPCPort           IntOrUndefined
	RPCHTTPPort       IntOrUndefined
	StatusCheck       BoolOrUndefined
	SyncRemoteCache   SyncRemoteCacheOption
	WaitForDeletions  WaitForDeletions
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

func (opts *SkaffoldOptions) IsTargetImage(artifact *latestV2.Artifact) bool {
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
