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

package kubectl

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func versionGetErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_KUBECTL_VERSION_ERR,
		})
}

func offlineModeErr() error {
	return sErrors.NewErrorWithStatusCode(
		proto.ActionableErr{
			Message: "cannot use offline mode if URL manifests are configured",
			ErrCode: proto.StatusCode_DEPLOY_KUBECTL_OFFLINE_MODE_ERR,
		})
}

func waitForDeletionErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("waiting for deletion: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_ERR_WAITING_FOR_DELETION,
		})
}

func readManifestErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
}

func readRemoteManifestErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_REMOTE_MANIFEST_ERR,
		})
}

func listManifestErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_LIST_MANIFEST_ERR,
		})
}

func userErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_KUBECTL_USER_ERR,
		})
}
