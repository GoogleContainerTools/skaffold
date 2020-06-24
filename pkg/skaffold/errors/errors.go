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
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto"
)

// These are phases in a DevLoop
const (
	Build       = Phase("Build")
	Deploy      = Phase("Deploy")
	StatusCheck = Phase("StatusCheck")
	FileSync    = Phase("FileSync")
	DevInit     = Phase("DevInit")
	Cleanup     = Phase("Cleanup")
)

var (
	// ErrNoSuggestionFound error not found
	ErrNoSuggestionFound = fmt.Errorf("no suggestions found")

	setOptionsOnce sync.Once
	skaffoldOpts   config.SkaffoldOptions
)

type Phase string

// SetSkaffoldOptions set Skaffold config options once. These options are used later to
// suggest actionable error messages based on skaffold run context
func SetSkaffoldOptions(opts config.SkaffoldOptions) {
	setOptionsOnce.Do(func() {
		skaffoldOpts = opts
	})
}

// ActionableErr returns an actionable error message with suggestions
func ActionableErr(phase Phase, err error) *proto.ActionableErr {
	errCode, suggestions := getErrorCodeFromError(phase, err)
	return &proto.ActionableErr{
		ErrCode:     errCode,
		Message:     err.Error(),
		Suggestions: suggestions,
	}
}

func ShowAIError(err error) error {
	for _, v := range knownBuildProblems {
		if v.regexp.MatchString(err.Error()) {
			if suggestions := v.suggestion(skaffoldOpts); suggestions != nil {
				return fmt.Errorf("%s. %s", v.description, concatSuggestions(suggestions))
			}
		}
	}
	return ErrNoSuggestionFound
}

func getErrorCodeFromError(phase Phase, err error) (proto.StatusCode, []*proto.Suggestion) {
	switch phase {
	case Build:
		for _, v := range knownBuildProblems {
			if v.regexp.MatchString(err.Error()) {
				return v.errCode, v.suggestion(skaffoldOpts)
			}
		}
		return proto.StatusCode_BUILD_UNKNOWN, nil
	case Deploy:
		return proto.StatusCode_DEPLOY_UNKNOWN, nil
	case StatusCheck:
		return proto.StatusCode_STATUSCHECK_UNKNOWN, nil
	case FileSync:
		return proto.StatusCode_SYNC_UNKNOWN, nil
	case DevInit:
		return proto.StatusCode_DEVINIT_UNKNOWN, nil
	case Cleanup:
		return proto.StatusCode_CLEANUP_UNKNOWN, nil
	}
	return proto.StatusCode_UNKNOWN_ERROR, nil
}

func concatSuggestions(suggestions []*proto.Suggestion) string {
	var s strings.Builder
	for _, suggestion := range suggestions {
		if s.String() != "" {
			s.WriteString(" or ")
		}
		s.WriteString(suggestion.Action)
	}
	s.WriteString(".")
	return s.String()
}
