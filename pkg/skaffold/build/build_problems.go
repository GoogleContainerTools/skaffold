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

package build

import (
	"context"
	"fmt"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

const (
	PushImageErr = "could not push image"
	// Error Prefix matches error thrown by Docker
	// See https://github.com/moby/moby/blob/master/client/errors.go#L18
	dockerConnectionFailed = ".*(Cannot connect to the Docker daemon.*) Is"

	// See https://github.com/moby/moby/blob/master/client/errors.go#L20
	// `docker build: error during connect: Post \"https://127.0.0.1:32770/v1.24/build?buildargs=:  globalRepo canceled`
	buildCancelled = ".*context canceled.*"
)

var (
	unknownProjectErr = fmt.Sprintf(".*(%s.* unknown: Project.*)", PushImageErr)
	nilSuggestions    = func(cfg interface{}) []*proto.Suggestion { return nil }
	// for testing
	getConfigForCurrentContext = config.GetConfigForCurrentKubectx
	problems                   = []sErrors.Problem{
		{
			Regexp:  re(fmt.Sprintf(".*%s.* denied: .*", PushImageErr)),
			ErrCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED,
			Description: func(err error) string {
				log.Entry(context.TODO()).Tracef("error building %s", err)
				return "Build Failed. No push access to specified image repository"
			},
			Suggestion: suggestBuildPushAccessDeniedAction,
		},
		{
			Regexp:  re(buildCancelled),
			ErrCode: proto.StatusCode_BUILD_CANCELLED,
			Description: func(error) string {
				return "Build Cancelled"
			},
			Suggestion: nilSuggestions,
		},
		{
			Regexp: re(unknownProjectErr),
			Description: func(err error) string {
				log.Entry(context.TODO()).Tracef("error building %s", err)
				matchExp := re(unknownProjectErr)
				if match := matchExp.FindStringSubmatch(err.Error()); len(match) >= 2 {
					return fmt.Sprintf("Build Failed. %s", match[1])
				}
				return fmt.Sprintf("Build Failed due to %s", err)
			},
			ErrCode: proto.StatusCode_BUILD_PROJECT_NOT_FOUND,
			Suggestion: func(interface{}) []*proto.Suggestion {
				return []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_GCLOUD_PROJECT,
					Action:         "Check your GCR project",
				}}
			},
		},
		{
			Regexp:  re(dockerConnectionFailed),
			ErrCode: proto.StatusCode_BUILD_DOCKER_DAEMON_NOT_RUNNING,
			Description: func(err error) string {
				log.Entry(context.TODO()).Tracef("error building %s", err)
				matchExp := re(dockerConnectionFailed)
				if match := matchExp.FindStringSubmatch(err.Error()); len(match) >= 2 {
					return fmt.Sprintf("Build Failed. %s", match[1])
				}
				return "Build Failed. Could not connect to Docker daemon"
			},
			Suggestion: func(interface{}) []*proto.Suggestion {
				return []*proto.Suggestion{{
					SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
					Action:         "Check if docker is running",
				}}
			},
		},
	}
)

// re is a shortcut around regexp.MustCompile
func re(s string) *regexp.Regexp {
	return regexp.MustCompile(s)
}

func init() {
	sErrors.AddPhaseProblems(constants.Build, problems)
}

func suggestBuildPushAccessDeniedAction(cfg interface{}) []*proto.Suggestion {
	buildCfg, ok := cfg.(Config)
	if !ok {
		return nil
	}
	if defaultRepo := buildCfg.DefaultRepo(); defaultRepo != nil {
		suggestions := []*proto.Suggestion{{
			SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO,
			Action:         "Check your `--default-repo` value",
		}}
		return append(suggestions, makeAuthSuggestionsForRepo(*defaultRepo))
	}

	// check if global repo is set
	if gCfg, err := getConfigForCurrentContext(buildCfg.GlobalConfig()); err == nil {
		if defaultRepo := gCfg.DefaultRepo; defaultRepo != "" {
			suggestions := []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_DEFAULT_REPO_GLOBAL_CONFIG,
				Action:         "Check your default-repo setting in skaffold config",
			}}
			return append(suggestions, makeAuthSuggestionsForRepo(defaultRepo))
		}
	}
	action := "Try running with `--default-repo` flag"
	if buildCfg.Mode() != config.RunModes.Build {
		// for `interactive` modes, also suggest user to use a local kubernetes cluster
		action += ". Otherwise start a local kubernetes cluster like `minikube`"
	}

	return []*proto.Suggestion{{
		SuggestionCode: proto.SuggestionCode_ADD_DEFAULT_REPO,
		Action:         action,
	}}
}

func makeAuthSuggestionsForRepo(repo string) *proto.Suggestion {
	// parse off the registry component; should have already been validated so unlikely to fail
	if ref, _ := docker.ParseReference(repo); ref != nil {
		repo = ref.Domain
	}

	if re(`(.+\.)?gcr\.io`).MatchString(repo) || re(`.+-docker\.pkg\.dev`).MatchString(repo) {
		return &proto.Suggestion{
			SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
			Action:         fmt.Sprintf("try `gcloud auth configure-docker%s`", withSpace(repo)),
		}
	}
	return &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_DOCKER_AUTH_CONFIGURE,
		Action:         fmt.Sprintf("try `docker login%s`", withSpace(repo)),
	}
}

// withSpace returns the given value with a space prepended when not empty.
func withSpace(value string) string {
	if len(value) > 0 {
		return " " + value
	}
	return value
}
