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
package validate

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

var (
	allowListedValidators = []string{"kubeval"}
	validatorAllowlist    = map[string]kptfile.Function{
		"kubeval": {Image: "gcr.io/kpt-fn/kubeval:v0.1"},
		// TODO: Add conftest validator in kpt catalog.
	}
)

// NewValidator instantiates a Validator object.
func NewValidator(config []latestV2.Validator) (*Validator, error) {
	var fns []kptfile.Function
	for _, c := range config {
		fn, ok := validatorAllowlist[c.Name]
		if !ok {
			return nil, errors.UnknownValidatorError(c.Name, allowListedValidators)
		}
		fns = append(fns, fn)
	}
	return &Validator{kptFn: fns}, nil
}

type Validator struct {
	kptFn []kptfile.Function
}

// GetDeclarativeValidators transforms and returns the skaffold validators defined in skaffold.yaml
func (v *Validator) GetDeclarativeValidators() []kptfile.Function {
	// TODO: guarantee the v.kptFn is updated once users changed skaffold.yaml file.
	return v.kptFn
}
