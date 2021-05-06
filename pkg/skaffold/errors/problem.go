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
	"regexp"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type descriptionFunc func(error) string
type suggestionFunc func(cfg interface{}) []*proto.Suggestion

// Problem defines a problem which can list suggestions and error codes
// evaluated when showing Actionable error messages
type Problem struct {
	Regexp      *regexp.Regexp
	Description func(error) string
	ErrCode     proto.StatusCode
	Suggestion  func(cfg interface{}) []*proto.Suggestion
	Err         error
}

func NewProblem(d descriptionFunc, sc proto.StatusCode, s suggestionFunc, err error) Problem {
	return Problem{
		Description: d,
		ErrCode:     sc,
		Suggestion:  s,
		Err:         err,
	}
}

func (p Problem) Error() string {
	description := fmt.Sprintf("%s.", p.Err)
	if p.Description != nil {
		description = p.Description(p.Err)
	}
	return description
}

func (p Problem) AIError(i interface{}, err error) error {
	p.Err = err
	if p.Suggestion == nil {
		return p
	}
	if suggestions := p.Suggestion(i); len(suggestions) > 0 {
		return fmt.Errorf("%s. %s", strings.Trim(p.Error(), "."), concatSuggestions(suggestions))
	}
	return p
}

func isProblem(err error) (Problem, bool) {
	if p, ok := err.(Problem); ok {
		return p, true
	}
	return Problem{}, false
}


type ProblemCatalog struct {
	allErrors map[constants.Phase][]Problem
	allErrorsLock sync.RWMutex
}

func (p ProblemCatalog) AddPhaseProblems(phase constants.Phase, problems []Problem) {
	p.allErrorsLock.Lock()
	if ps, ok := p.allErrors[phase]; ok {
		problems = append(ps, problems...)
	}
	p.allErrors[phase] = problems
	p.allErrorsLock.Unlock()
}

func (p ProblemCatalog) GetProblemCatalog() ProblemCatalog {
	return copy(p)
}

func NewCatalog() ProblemCatalog {
	return ProblemCatalog{}
}