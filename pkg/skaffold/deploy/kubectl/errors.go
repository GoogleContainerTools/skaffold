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

	deployerr "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/error"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

const (
	toolName    = "Kubectl"
	installLink = "https://kubernetes.io/docs/tasks/tools/install-kubectl"
)

func versionGetErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: deployerr.MissingToolErr(toolName, err),
			ErrCode: proto.StatusCode_DEPLOY_KUBECTL_VERSION_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INSTALL_KUBECTL,
					Action:         fmt.Sprintf("Please install kubectl via %s", installLink),
				},
			},
		})
}

func waitForDeletionErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("waiting for deletion: %s", err),
			ErrCode: proto.StatusCode_DEPLOY_ERR_WAITING_FOR_DELETION,
		})
}

func readManifestErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_DEPLOY_READ_MANIFEST_ERR,
		})
}

func userErr(err error) error {
	return deployerr.UserError(err, proto.StatusCode_DEPLOY_KUBECTL_USER_ERR)
}
