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

package docker

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	noSpaceLeft   = regexp.MustCompile(".*no space left.*")
	execFormatErr = regexp.MustCompile(".*exec format error.*")
)

// newBuildError turns Docker-specific errors into actionable errors.
// The input errors are assumed to be from the Skaffold docker invocation.
func newBuildError(err error, cfg docker.Config) error {
	if sErrors.IsSkaffoldErr(err) {
		return err
	}
	errU := errors.Unwrap(err)
	if errU == nil {
		return err
	}

	switch errU.(type) {
	case *jsonmessage.JSONError:
		return sErrors.NewError(err,
			&proto.ActionableErr{
				Message: err.Error(),
				ErrCode: proto.StatusCode_BUILD_USER_ERROR,
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_FIX_USER_BUILD_ERR,
						Action:         "Please fix the Dockerfile and try again.",
					},
				},
			})
	default:
		return sErrors.NewError(err, getActionableErr(errU, cfg))
	}
}

func getActionableErr(err error, cfg docker.Config) *proto.ActionableErr {
	var errCode proto.StatusCode
	suggestions := []*proto.Suggestion{
		{
			SuggestionCode: proto.SuggestionCode_DOCKER_BUILD_RETRY,
			Action:         "Docker build ran into internal error. Please retry.\nIf this keeps happening, please open an issue.",
		},
	}
	switch err.(type) {
	case errdefs.ErrNotFound:
		errCode = proto.StatusCode_BUILD_DOCKER_ERROR_NOT_FOUND
	case errdefs.ErrInvalidParameter:
		errCode = proto.StatusCode_BUILD_DOCKER_INVALID_PARAM_ERR
	case errdefs.ErrConflict:
		errCode = proto.StatusCode_BUILD_DOCKER_CONFLICT_ERR
	case errdefs.ErrCancelled:
		errCode = proto.StatusCode_BUILD_DOCKER_CANCELLED
	case errdefs.ErrForbidden:
		errCode = proto.StatusCode_BUILD_DOCKER_FORBIDDEN_ERR
	case errdefs.ErrDataLoss:
		errCode = proto.StatusCode_BUILD_DOCKER_DATA_LOSS_ERR
	case errdefs.ErrDeadline:
		errCode = proto.StatusCode_BUILD_DOCKER_DEADLINE
	case errdefs.ErrNotImplemented:
		errCode = proto.StatusCode_BUILD_DOCKER_NOT_IMPLEMENTED_ERR
	case errdefs.ErrNotModified:
		errCode = proto.StatusCode_BUILD_DOCKER_NOT_MODIFIED_ERR
	case errdefs.ErrSystem:
		errCode = proto.StatusCode_BUILD_DOCKER_SYSTEM_ERR
		if noSpaceLeft.MatchString(err.Error()) {
			errCode = proto.StatusCode_BUILD_DOCKER_NO_SPACE_ERR
			suggestions = []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_RUN_DOCKER_PRUNE,
					Action:         "Docker ran out of memory. Please run 'docker system prune' to removed unused docker data",
				},
			}
			if !cfg.Prune() && (cfg.Mode() == config.RunModes.Dev || cfg.Mode() == config.RunModes.Debug) {
				suggestions = append(suggestions, &proto.Suggestion{
					SuggestionCode: proto.SuggestionCode_SET_CLEANUP_FLAG,
					Action:         fmt.Sprintf("Run skaffold %s with --cleanup=true to clean up images built by skaffold", cfg.Mode()),
				})
			}
		}
	case errdefs.ErrUnauthorized:
		errCode = proto.StatusCode_BUILD_DOCKER_UNAUTHORIZED
	case errdefs.ErrUnavailable:
		errCode = proto.StatusCode_BUILD_DOCKER_UNAVAILABLE
	default:
		errCode = proto.StatusCode_BUILD_DOCKER_UNKNOWN
	}

	return &proto.ActionableErr{
		Message:     err.Error(),
		ErrCode:     errCode,
		Suggestions: suggestions,
	}
}

func dockerfileNotFound(err error, artifact string) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_DOCKERFILE_NOT_FOUND,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_SKAFFOLD_CONFIG_DOCKERFILE,
					Action: fmt.Sprintf("Dockerfile not found. Please check config `dockerfile` for artifact %s."+
						"\nRefer https://skaffold.dev/docs/references/yaml/#build-artifacts-docker for details.", artifact),
				},
			},
		})
}

func cacheFromPullErr(err error, artifact string) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_DOCKER_CACHE_FROM_PULL_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_CACHE_FROM_ARTIFACT_CONFIG,
					Action: fmt.Sprintf("Fix `cacheFrom` config for artifact %s."+
						"\nRefer https://skaffold.dev/docs/references/yaml/#build-artifacts-docker for details.", artifact),
				},
			},
		})
}

func tryExecFormatErr(err error, stdErr bytes.Buffer) error {
	if !execFormatErr.MatchString(stdErr.String()) {
		return err
	}
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_CROSS_PLATFORM_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_BUILD_INSTALL_PLATFORM_EMULATORS,
					Action:         "To run cross-platform builds, check that QEMU platform emulators are installed correctly. To install, run:\n\n\tdocker run --privileged --rm tonistiigi/binfmt --install all\n\nFor more details, see https://registry.hub.docker.com/r/tonistiigi/binfmt",
				},
			},
		})
}

func tryExecFormatErrBuildX(err error, stdErr bytes.Buffer) error {
	if !execFormatErr.MatchString(stdErr.String()) {
		return err
	}
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_CROSS_PLATFORM_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_BUILD_INSTALL_PLATFORM_EMULATORS,
					Action:         "To run cross-platform builds, use a proper buildx builder. To create and select it, run:\n\n\tdocker buildx create --driver docker-container --name buildkit\n\n\tskaffold config set buildx-builder buildkit\n\nFor more details, see https://docs.docker.com/build/building/multi-platform/",
				},
			},
		})
}
