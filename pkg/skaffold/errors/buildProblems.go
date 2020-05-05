/*
Copyright 2020 The Skaffold Authors

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

package errors

import (
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

var (
	// for testing
	getConfigForCurrentContext = config.GetConfigForCurrentKubectx
)

func suggestBuildPushAccessDeniedAction(opts config.SkaffoldOptions) string {
	action := "Trying running with `--default-repo` flag."
	if opts.DefaultRepo.Value() != nil {
		return curateErrorMessage(*opts.DefaultRepo.Value(), "Check your `--default-repo` value")
	}
	// check if global repo is set
	if cfg, err := getConfigForCurrentContext(opts.GlobalConfig); err == nil {
		if cfg.DefaultRepo != "" {
			return curateErrorMessage(cfg.DefaultRepo, "Check your default-repo setting in skaffold config")
		}
	}
	return action
}

func curateErrorMessage(repo string, prefix string) string {
	if strings.HasPrefix(repo, "gcr.io") {
		return prefix + " or try `gcloud auth configure-docker`."
	}
	return prefix + " or try `docker login`."
}
