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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

var (
	// GetProblemCatalogCopy get a copies of the current problem catalog.
	GetProblemCatalogCopy = getProblemCatalogCopy
)

type ProblemCatalog struct {
	allErrors map[constants.Phase][]Problem
}

func (p ProblemCatalog) AddPhaseProblems(phase constants.Phase, problems []Problem) {
	if ps, ok := p.allErrors[phase]; ok {
		problems = append(ps, problems...)
	}
	p.allErrors[phase] = problems
}

func getProblemCatalogCopy() ProblemCatalog {
	return ProblemCatalog{
		allErrors: problemCatalog.allErrors,
	}
}

func NewProblemCatalog() ProblemCatalog {
	problemCatalog = ProblemCatalog{
		allErrors: map[constants.Phase][]Problem{},
	}
	return problemCatalog
}
