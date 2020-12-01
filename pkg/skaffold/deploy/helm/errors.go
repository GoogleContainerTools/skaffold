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

package helm

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func versionGetErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf(versionErrorString, err.Error()),
			ErrCode: proto.StatusCode_DEPLOY_HELM_VERSION_ERR,
		})
}

func minVersionErr() error {
	return sErrors.NewErrorWithStatusCode(
		proto.ActionableErr{
			Message: "skaffold requires Helm version 3.0.0-beta.0 or greater",
			ErrCode: proto.StatusCode_DEPLOY_HELM_MIN_VERSION_ERR,
		})
}

func helmLabelErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_HELM_APPLY_LABELS,
		})
}

func userErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_HELM_USER_ERR,
		})
}

func noMatchingBuild(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("matching build results to chart values: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_NO_MATCHING_BUILD,
		})
}
