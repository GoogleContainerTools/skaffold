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
	"errors"
	"fmt"

	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto"
)

// newBuildError turns Docker-specific errors into actionable errors.
// The input errors are assumed to be from the Skaffold docker invocation.
func newBuildError(err error) error {
	errU := errors.Unwrap(err)
	if errU == nil {
		return err
	}

	switch errU.(type) {
	case *jsonmessage.JSONError:
		return sErrors.NewError(err,
			proto.ActionableErr{
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
		return sErrors.NewError(err,
			proto.ActionableErr{
				Message: errU.Error(),
				ErrCode: getErrorCode(errU),
				Suggestions: []*proto.Suggestion{
					{
						SuggestionCode: proto.SuggestionCode_DOCKER_BUILD_RETRY,
						Action:         "Docker build ran into internal error. Please retry.\nIf this keeps happening, please open an issue.",
					},
				},
			})
	}
}

func getErrorCode(err error) proto.StatusCode {
	switch err.(type) {
	case errdefs.ErrNotFound:
		return proto.StatusCode_BUILD_DOCKER_ERROR_NOT_FOUND
	case errdefs.ErrInvalidParameter:
		return proto.StatusCode_BUILD_DOCKER_INVALID_PARAM_ERR
	case errdefs.ErrConflict:
		return proto.StatusCode_BUILD_DOCKER_CONFLICT_ERR
	case errdefs.ErrCancelled:
		return proto.StatusCode_BUILD_DOCKER_CANCELLED
	case errdefs.ErrForbidden:
		return proto.StatusCode_BUILD_DOCKER_FORBIDDEN_ERR
	case errdefs.ErrDataLoss:
		return proto.StatusCode_BUILD_DOCKER_DATA_LOSS_ERR
	case errdefs.ErrDeadline:
		return proto.StatusCode_BUILD_DOCKER_DEADLINE
	case errdefs.ErrNotImplemented:
		return proto.StatusCode_BUILD_DOCKER_NOT_IMPLEMENTED_ERR
	case errdefs.ErrNotModified:
		return proto.StatusCode_BUILD_DOCKER_NOT_MODIFIED_ERR
	case errdefs.ErrSystem:
		return proto.StatusCode_BUILD_DOCKER_SYSTEM_ERR
	case errdefs.ErrUnauthorized:
		return proto.StatusCode_BUILD_DOCKER_UNAUTHORIZED
	case errdefs.ErrUnavailable:
		return proto.StatusCode_BUILD_DOCKER_UNAVAILABLE
	default:
		return proto.StatusCode_BUILD_DOCKER_UNKNOWN
	}
}

func dockerfileNotFound(err error, artifact string) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
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
		proto.ActionableErr{
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
