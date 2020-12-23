/*
Copyright 2019 The Skaffold Authors

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

package jib

import (
	"fmt"

	"github.com/sirupsen/logrus"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto"
)

func unknownPluginType(ws string) error {
	return sErrors.NewErrorWithStatusCode(
		proto.ActionableErr{
			Message: fmt.Sprintf("Unknown Jib builder type for workspace %s", ws),
			ErrCode: proto.StatusCode_BUILD_UNKNOWN_JIB_PLUGIN_TYPE,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_JIB_PLUGIN_CONFIGURATION,
					Action:         fmt.Sprintf("Use one of supported Jib plugin types [%s, %s]", JibMaven, JibGradle),
				},
			},
		})
}

func unableToDeterminePluginType(ws string, err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("unable to determine Jib builder type for workspace %s due to %s", ws, err),
			ErrCode: proto.StatusCode_BUILD_UNKNOWN_JIB_PLUGIN_TYPE,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_JIB_PLUGIN_CONFIGURATION,
					Action:         fmt.Sprintf("Use one of supported Jib plugin types [%s, %s]", JibMaven, JibGradle),
				},
			},
		})
}

func dependencyErr(pType PluginType, workspace string, err error) error {
	var code proto.StatusCode
	switch pType {
	case JibMaven:
		code = proto.StatusCode_BUILD_JIB_MAVEN_DEP_ERR
	case JibGradle:
		code = proto.StatusCode_BUILD_JIB_GRADLE_DEP_ERR
	default:
		logrus.Fatal("Unknown jib build type", pType)
	}
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: fmt.Sprintf("could not fetch dependencies for workspace %s: %s", workspace, err.Error()),
			ErrCode: code,
		})
}

func jibToolErr(err error) error {
	return sErrors.NewError(err,
		proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_USER_ERROR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_FIX_USER_BUILD_ERR,
					Action:         "See the build transcript for suggestions.",
				},
			},
		})
}
