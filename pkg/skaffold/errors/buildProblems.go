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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto"
)

const (
	PushImageErrPrefix = "could not push image"
)

var (
	// for testing
	getConfigForCurrentContext = config.GetConfigForCurrentKubectx
)

func suggestBuildPushAccessDeniedAction(opts config.SkaffoldOptions) []*proto.Suggestion {
	if defaultRepo := opts.DefaultRepo.Value(); defaultRepo != nil {
		suggestions := []*proto.Suggestion{{
			SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO,
			Action:         "Check your `--default-repo` value",
		}}
		return append(suggestions, makeAuthSuggestionsForRepo(*defaultRepo))
	}

	// check if global repo is set
	if cfg, err := getConfigForCurrentContext(opts.GlobalConfig); err == nil {
		if defaultRepo := cfg.DefaultRepo; defaultRepo != "" {
			suggestions := []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO_GLOBAL_CONFIG,
				Action:         "Check your default-repo setting in skaffold config",
			}}
			return append(suggestions, makeAuthSuggestionsForRepo(defaultRepo))
		}
	}

	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_ADD_DEFAULT_REPO,
		Action:         "Trying running with `--default-repo` flag",
	}}
}

func makeAuthSuggestionsForRepo(repo string) *proto.Suggestion {
	if re(`(.+\.)?gcr\.io.*`).MatchString(repo) {
		return &proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
			Action:         "try `gcloud auth configure-docker`",
		}
	}
	return &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_DOCKER_AUTH_CONFIGURE,
		Action:         "try `docker login`",
	}
}
