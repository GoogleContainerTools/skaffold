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
	"context"
	"fmt"
	"os/exec"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

var (
	allowListedValidators = []string{"kubeval"}
	validatorAllowlist    = map[string]kptfile.Function{
		"kubeval": {Image: "gcr.io/kpt-fn/kubeval:v0.1"},
		// TODO: Add conftest validator in kpt catalog.
		"gatekeeper": {
			Image:     "gcr.io/kpt-fn/gatekeeper:v0.2.1",
			ConfigMap: map[string]string{}},
	}
)

// NewValidator instantiates a Validator object.
func NewValidator(config []latest.Validator) (Validator, error) {
	var fns []kptfile.Function
	for _, c := range config {
		fn, ok := validatorAllowlist[c.Name]
		if !ok {
			// TODO: Add links to explain "skaffold-managed mode" and "kpt-managed mode".
			return Validator{}, sErrors.NewErrorWithStatusCode(
				&proto.ActionableErr{
					Message: fmt.Sprintf("unsupported validator %q", c.Name),
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
		fns = append(fns, fn)
	}
	return Validator{kptFn: fns}, nil
}

type Validator struct {
	kptFn []kptfile.Function
}

// GetDeclarativeValidators transforms and returns the skaffold validators defined in skaffold.yaml
func (v Validator) GetDeclarativeValidators() []kptfile.Function {
	// TODO: guarantee the v.kptFn is updated once users changed skaffold.yaml file.
	return v.kptFn
}

func (v Validator) Validate(ctx context.Context, ml manifest.ManifestList) error {
	if v.kptFn == nil || len(v.kptFn) == 0 {
		return nil
	}

	for _, validator := range v.kptFn {
		kvs := util.EnvMapToSlice(validator.ConfigMap, "=")
		args := []string{"fn", "eval", "-i", validator.Image, "-o", "unwrap", "-", "--"}
		args = append(args, kvs...)
		cmd := exec.CommandContext(ctx, "kpt", args...)
		reader := ml.Reader()
		cmd.Stdin = reader
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}
