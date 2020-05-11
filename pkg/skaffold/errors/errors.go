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
)

type Phase string

func ErrorCodeFromError(phase Phase, _ error) proto.StatusCode {
	switch phase {
	case Build:
		return proto.StatusCode_BUILD_UNKNOWN
	case Deploy:
		return proto.StatusCode_DEPLOY_UNKNOWN
	case StatusCheck:
		return proto.StatusCode_STATUSCHECK_UNKNOWN
	case FileSync:
		return proto.StatusCode_SYNC_UNKNOWN
	case DevInit:
		return proto.StatusCode_DEVINIT_UNKNOWN
	case Cleanup:
		return proto.StatusCode_CLEANUP_UNKNOWN
	}
	return proto.StatusCode_UNKNOWN_ERROR
}

func ShowAIError(err error, opts config.SkaffoldOptions) error {
	for _, v := range knownBuildProblems {
		if v.regexp.MatchString(err.Error()) {
			if s := v.suggestion(opts); s != "" {
				return fmt.Errorf("%s. %s", v.description, s)
			}
		}
	}
	return ErrNoSuggestionFound
}
