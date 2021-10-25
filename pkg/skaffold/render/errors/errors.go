/*
Copyright 2021 The Skaffold Authors

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

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

// DeleteKptfileError returns the error when unable to delete a Kptfile.
func DeleteKptfileError(err error, hydrationDir string) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("unable to delete Kptfile in %v", hydrationDir),
			ErrCode: proto.StatusCode_RENDER_KPTFILE_INIT_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_MANUAL_INIT,
					Action:         fmt.Sprintf("suggest manually delete `rm -rf %v`", hydrationDir),
				},
			},
		})
}

// ParseKptfileError returns the error when unable to parse the Kptfile.
func ParseKptfileError(err error, hydrationDir string) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("unable to parse Kptfile in %v", hydrationDir),
			ErrCode: proto.StatusCode_RENDER_KPTFILE_INVALID_YAML_ERR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_KPTFILE_CHECK_YAML,
					Action: fmt.Sprintf("please check if the Kptfile is correct and " +
						"the `apiVersion` is greater than `v1alpha2`"),
				},
			},
		})
}

// UnknownTransformerError returns the error when user provides an unsupported transformer.
func UnknownTransformerError(transformerName string, allowListedTransformer []string) error {
	// TODO: Add links to explain "skaffold-managed mode" and "kpt-managed mode".
	return sErrors.NewErrorWithStatusCode(
		&proto.ActionableErr{
			Message: fmt.Sprintf("unsupported transformer %q", transformerName),
			ErrCode: proto.StatusCode_CONFIG_UNKNOWN_TRANSFORMER,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_ALLOWLIST_transformers,
					Action: fmt.Sprintf(
						"please only use the following transformers in skaffold-managed mode: %v. "+
							"to use custom transformers, please use kpt-managed mode.",
						allowListedTransformer),
				},
			},
		})
}

// BadTransformerParamsError returns the error when user provides incorrect ConfigMap to the transform function.
func BadTransformerParamsError(transformerName string) error {
	return sErrors.NewErrorWithStatusCode(
		&proto.ActionableErr{
			Message: fmt.Sprintf("unknown arguments for transformer %v", transformerName),
			ErrCode: proto.StatusCode_CONFIG_UNKNOWN_TRANSFORMER,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_ALLOWLIST_transformers,
					Action: fmt.Sprintf("please check if the .transformer field and " +
						"make sure `configMapData` is a list of data in the form of `${KEY}=${VALUE}`"),
				},
			},
		})
}

// UnknownValidatorError returns the error when user provides an unsupported validator.
func UnknownValidatorError(validatorName string, allowListedValidators []string) error {
	// TODO: Add links to explain "skaffold-managed mode" and "kpt-managed mode".
	return sErrors.NewErrorWithStatusCode(
		&proto.ActionableErr{
			Message: fmt.Sprintf("unsupported validator %q", validatorName),
			ErrCode: proto.StatusCode_CONFIG_UNKNOWN_VALIDATOR,
			Suggestions: []*proto.Suggestion{
				{
					SuggestionCode: proto.SuggestionCode_CONFIG_ALLOWLIST_VALIDATORS,
					Action: fmt.Sprintf(
						"please only use the following validators in skaffold-managed mode: %v. "+
							"to use custom validators, please use kpt-managed mode.", allowListedValidators),
				},
			},
		})
}
