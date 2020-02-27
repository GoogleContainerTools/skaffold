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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// PortForwardOptions are options set by the command line for port forwarding
// with additional configuration information as well
type PortForwardOptions struct {
	Enabled     bool
	ForwardPods bool
}

// SkaffoldOptions are options that are set by command line arguments not included
// in the config file itself
type SkaffoldOptions struct {
	ConfigurationFile  string
	GlobalConfig       string
	Cleanup            bool
	Notification       bool
	Tail               bool
	TailDev            bool
	SkipTests          bool
	CacheArtifacts     bool
	EnableRPC          bool
	Force              bool
	NoPrune            bool
	NoPruneChildren    bool
	StatusCheck        bool
	AutoBuild          bool
	AutoSync           bool
	AutoDeploy         bool
	RenderOnly         bool
	PortForward        PortForwardOptions
	CustomTag          string
	Namespace          string
	CacheFile          string
	Trigger            string
	KubeContext        string
	KubeConfig         string
	WatchPollInterval  int
	DefaultRepo        string
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
}

// Labels returns a map of labels to be applied to all deployed
// k8s objects during the duration of the run
func (opts *SkaffoldOptions) Labels() map[string]string {
	labels := map[string]string{}

	if opts.Cleanup {
		labels["skaffold.dev/cleanup"] = "true"
	}
	if opts.Tail || opts.TailDev {
		labels["skaffold.dev/tail"] = "true"
	}
	if opts.Namespace != "" {
		labels["skaffold.dev/namespace"] = opts.Namespace
	}
	for i, profile := range opts.Profiles {
		key := fmt.Sprintf("skaffold.dev/profile.%d", i)
		labels[key] = profile
	}
	for _, cl := range opts.CustomLabels {
		l := strings.SplitN(cl, "=", 2)
		if len(l) == 1 {
			labels[l[0]] = ""
			continue
		}
		labels[l[0]] = l[1]
	}
	return labels
}

// Prune returns true iff the user did NOT specify the --no-prune flag,
// and the user did NOT specify the --cache-artifacts flag.
func (opts *SkaffoldOptions) Prune() bool {
	return !opts.NoPrune && !opts.CacheArtifacts
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
