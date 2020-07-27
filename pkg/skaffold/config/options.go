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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// PortForwardOptions are options set by the command line for port forwarding
// with additional configuration information as well
type PortForwardOptions struct {
	Enabled     bool
	ForwardPods bool
}

// WaitForDeletions configures the wait for pending deletions.
type WaitForDeletions struct {
	Max     time.Duration
	Delay   time.Duration
	Enabled bool
}

// SkaffoldOptions are options that are set by command line arguments not included
// in the config file itself
type SkaffoldOptions struct {
	ConfigurationFile     string
	GlobalConfig          string
	Cleanup               bool
	Notification          bool
	Tail                  bool
	SkipTests             bool
	CacheArtifacts        bool
	EnableRPC             bool
	Force                 bool
	NoPrune               bool
	NoPruneChildren       bool
	StatusCheck           bool
	AutoBuild             bool
	AutoSync              bool
	AutoDeploy            bool
	RenderOnly            bool
	RenderOutput          string
	ProfileAutoActivation bool
	DryRun                bool
	SkipRender            bool

	// Add Skaffold-specific labels including runID, deployer labels, etc.
	// `CustomLabels` are still applied if this is false. Must only be used in
	// commands which don't deploy (e.g. `skaffold render`) since the runID
	// label isn't available.
	AddSkaffoldLabels bool

	PortForward        PortForwardOptions
	CustomTag          string
	Namespace          string
	CacheFile          string
	Trigger            string
	KubeContext        string
	KubeConfig         string
	DigestSource       string
	WatchPollInterval  int
	DefaultRepo        StringOrUndefined
	CustomLabels       []string
	TargetImages       []string
	Profiles           []string
	InsecureRegistries []string
	Command            string
	RPCPort            int
	RPCHTTPPort        int

	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles
	MinikubeProfile string
	SuppressLogs    []string

	WaitForDeletions WaitForDeletions
}

// Prune returns true iff the user did NOT specify the --no-prune flag,
// and the user did NOT specify the --cache-artifacts flag.
func (opts *SkaffoldOptions) Prune() bool {
	return !opts.NoPrune && !opts.CacheArtifacts
}

func (opts *SkaffoldOptions) IsDevMode() bool {
	return opts.Command == "dev"
}

func (opts *SkaffoldOptions) IsDebugMode() bool {
	return opts.Command == "debug"
}

func (opts *SkaffoldOptions) IsTargetImage(artifact *latest.Artifact) bool {
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
