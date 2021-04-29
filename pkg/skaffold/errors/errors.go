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
	"errors"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	protoV2 "github.com/GoogleContainerTools/skaffold/proto/v2"
)

const (
	// Report issue text
	reportIssueText = "If above error is unexpected, please open an issue to report this error at " + constants.GithubIssueLink

	// PushImageErr is the error prepended.
	PushImageErr = "could not push image"
)

var (
	ReportIssueSuggestion = func(interface{}) []*proto.Suggestion {
		return []*proto.Suggestion{{
			SuggestionCode: proto.SuggestionCode_OPEN_ISSUE,
			Action:         reportIssueText,
		}}
	}
)

// ActionableErr returns an actionable error message with suggestions
func ActionableErr(cfg interface{}, phase constants.Phase, err error) *proto.ActionableErr {
	errCode, suggestions := getErrorCodeFromError(cfg, phase, err)
	return &proto.ActionableErr{
		ErrCode:     errCode,
		Message:     err.Error(),
		Suggestions: suggestions,
	}
}

// ActionableErrV2 returns an actionable error message with suggestions
func ActionableErrV2(cfg interface{}, phase constants.Phase, err error) *protoV2.ActionableErr {
	errCode, suggestions := getErrorCodeFromError(cfg, phase, err)
	suggestionsV2 := make([]*protoV2.Suggestion, len(suggestions))
	for i, suggestion := range suggestions {
		converted := protoV2.Suggestion(*suggestion)
		suggestionsV2[i] = &converted
	}
	return &protoV2.ActionableErr{
		ErrCode:     errCode,
		Message:     err.Error(),
		Suggestions: suggestionsV2,
	}
}

func ShowAIError(cfg interface{}, err error) error {
	if uErr := errors.Unwrap(err); uErr != nil {
		err = uErr
	}
	if IsSkaffoldErr(err) {
		instrumentation.SetErrorCode(err.(Error).StatusCode())
		return err
	}

	if p, ok := isProblem(err); ok {
		instrumentation.SetErrorCode(p.ErrCode)
		return p.AIError(cfg, err)
	}

	allErrorsLock.RLock()
	defer allErrorsLock.RUnlock()
	for _, problems := range allErrors {
		for _, p := range problems {
			if p.Regexp.MatchString(err.Error()) {
				instrumentation.SetErrorCode(p.ErrCode)
				return p.AIError(cfg, err)
			}
		}
	}
	return err
}

func getErrorCodeFromError(cfg interface{}, phase constants.Phase, err error) (proto.StatusCode, []*proto.Suggestion) {
	var sErr Error
	if errors.As(err, &sErr) {
		return sErr.StatusCode(), sErr.Suggestions()
	}

	if problems, ok := allErrors[phase]; ok {
		for _, p := range problems {
			if p.Regexp.MatchString(err.Error()) {
				return p.ErrCode, p.Suggestion(cfg)
			}
		}
	}
	return unknownErrForPhase(phase), ReportIssueSuggestion(cfg)
}

func concatSuggestions(suggestions []*proto.Suggestion) string {
	var s strings.Builder
	for _, suggestion := range suggestions {
		if s.String() != "" {
			s.WriteString(" or ")
		}
		s.WriteString(suggestion.Action)
	}
	if s.String() == "" {
		return ""
	}
	s.WriteString(".")
	return s.String()
}

func unknownErrForPhase(phase constants.Phase) proto.StatusCode {
	switch phase {
	case constants.Build:
		return proto.StatusCode_BUILD_UNKNOWN
	case constants.Init:
		return proto.StatusCode_INIT_UNKNOWN
	case constants.Test:
		return proto.StatusCode_TEST_UNKNOWN
	case constants.Deploy:
		return proto.StatusCode_DEPLOY_UNKNOWN
	case constants.StatusCheck:
		return proto.StatusCode_STATUSCHECK_UNKNOWN
	case constants.Sync:
		return proto.StatusCode_SYNC_UNKNOWN
	case constants.DevInit:
		return proto.StatusCode_DEVINIT_UNKNOWN
	case constants.Cleanup:
		return proto.StatusCode_CLEANUP_UNKNOWN
	default:
		return proto.StatusCode_UNKNOWN_ERROR
	}
}
