/*
Copyright 2022 The Skaffold Authors

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

	"github.com/pkg/errors"

	deployerr "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/error"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	// error to throw when helm version can't be determined
	versionErrorString = "failed to determine binary version: %w"
)

const (
	installLink = "https://helm.sh/docs/intro/install"
	toolName    = "Helm"
)

func VersionGetErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: deployerr.MissingToolErr(toolName, fmt.Errorf(versionErrorString, err)),
			ErrCode: proto.StatusCode_DEPLOY_HELM_VERSION_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_INSTALL_HELM,
					Action:         fmt.Sprintf("Please install helm via %s", installLink),
				},
			},
		})
}

func MinVersionErr(minVer string) error {
	return sErrors.NewErrorWithStatusCode(
		&proto.ActionableErr{
			Message: fmt.Sprintf("skaffold requires Helm version %s or greater", minVer),
			ErrCode: proto.StatusCode_DEPLOY_HELM_MIN_VERSION_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_UPGRADE_HELM,
					Action:         fmt.Sprintf("Please upgrade helm to %s or higher via %s", minVer, installLink),
				},
			},
		})
}

func UserErr(prefix string, err error) error {
	return deployerr.UserError(errors.Wrap(err, prefix), proto.StatusCode_DEPLOY_HELM_USER_ERR)
}

func CreateNamespaceErr(version string) error {
	return sErrors.NewErrorWithStatusCode(
		&proto.ActionableErr{
			Message: fmt.Sprintf("Skaffold config options `createNamespace` is not available in the current Helm version %s", version),
			ErrCode: proto.StatusCode_DEPLOY_HELM_CREATE_NS_NOT_AVAILABLE,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_UPGRADE_HELM32,
					Action:         "\nPlease update Helm to version 3.2 or higher",
				},
				{
					SuggestionCode: proto.SuggestionCode_FIX_SKAFFOLD_CONFIG_HELM_CREATE_NAMESPACE,
					Action:         "set `releases.createNamespace` to false and try again",
				},
			},
		})
}
