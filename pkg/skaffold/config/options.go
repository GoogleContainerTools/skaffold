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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// SkaffoldOptions are options that are set by command line arguments not included
// in the config file itself
type SkaffoldOptions struct {
	ConfigurationFile  string
	Cleanup            bool
	Notification       bool
	Tail               bool
	TailDev            bool
	PortForward        bool
	SkipTests          bool
	CacheArtifacts     bool
	EnableRPC          bool
	Force              bool
	ForceDev           bool
	NoPrune            bool
	NoPruneChildren    bool
	CustomTag          string
	Namespace          string
	CacheFile          string
	Trigger            string
	WatchPollInterval  int
	DefaultRepo        string
	CustomLabels       []string
	TargetImages       []string
	Profiles           []string
	InsecureRegistries []string
	Command            string
	RPCPort            int
	RPCHTTPPort        int
	StatusCheck        bool
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
	if len(opts.Profiles) > 0 {
		labels["skaffold.dev/profiles"] = strings.Join(opts.Profiles, "__")
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

func (opts *SkaffoldOptions) ForceDeploy() bool {
	return opts.ForceDev || opts.Force
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
