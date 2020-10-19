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
	"fmt"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto"
)

// re is a shortcut around regexp.MustCompile
func re(s string) *regexp.Regexp {
	return regexp.MustCompile(s)
}

type problem struct {
	regexp      *regexp.Regexp
	description func(error) string
	errCode     proto.StatusCode
	suggestion  func(opts config.SkaffoldOptions) []*proto.Suggestion
}

// Build Problems are Errors in build phase
var knownBuildProblems = []problem{
	{
		regexp:  re(fmt.Sprintf(".*%s.* denied: .*", PushImageErr)),
		errCode: proto.StatusCode_BUILD_PUSH_ACCESS_DENIED,
		description: func(error) string {
			return "Build Failed. No push access to specified image repository"
		},
		suggestion: suggestBuildPushAccessDeniedAction,
	},
	{
		regexp:  re(BuildCancelled),
		errCode: proto.StatusCode_BUILD_CANCELLED,
		description: func(error) string {
			return "Build Cancelled."
		},
		suggestion: func(config.SkaffoldOptions) []*proto.Suggestion {
			return nil
		},
	},
	{
		regexp: re(fmt.Sprintf(".*%s.* unknown: Project", PushImageErr)),
		description: func(error) string {
			return "Build Failed"
		},
		errCode: proto.StatusCode_BUILD_PROJECT_NOT_FOUND,
		suggestion: func(config.SkaffoldOptions) []*proto.Suggestion {
			return []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_GCLOUD_PROJECT,
				Action:         "Check your GCR project",
			}}
		},
	},
	{
		regexp:  re(DockerConnectionFailed),
		errCode: proto.StatusCode_BUILD_DOCKER_DAEMON_NOT_RUNNING,
		description: func(err error) string {
			matchExp := re(DockerConnectionFailed)
			if match := matchExp.FindStringSubmatch(fmt.Sprintf("%s", err)); len(match) >= 2 {
				return fmt.Sprintf("Build Failed. %s", match[1])
			}
			return "Build Failed. Could not connect to Docker daemon"
		},
		suggestion: func(config.SkaffoldOptions) []*proto.Suggestion {
			return []*proto.Suggestion{{
				SuggestionCode: proto.SuggestionCode_CHECK_DOCKER_RUNNING,
				Action:         "Check if docker is running",
			}}
		},
	},
}
