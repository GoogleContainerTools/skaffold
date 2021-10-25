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
package transform

import (
	"fmt"
	"strings"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var (
	allowListedTransformer = []string{"set-labels"}
	transformerAllowlist   = map[string]kptfile.Function{
		"set-namespace": {
			Image:     "gcr.io/kpt-fn/set-namespace",
			ConfigMap: map[string]string{},
		},
		"set-labels": {
			Image:     "gcr.io/kpt-fn/set-labels:v0.1",
			ConfigMap: map[string]string{},
		},
		"set-annotations": {
			Image:     "gcr.io/kpt-fn/set-annotations:v0.1",
			ConfigMap: map[string]string{},
		},
		"create-setters": {
			Image:     "gcr.io/kpt-fn/create-setters:unstable",
			ConfigMap: map[string]string{},
		},
		"apply-setters": {
			Image:     "gcr.io/kpt-fn/apply-setters:unstable",
			ConfigMap: map[string]string{},
		},
	}

	AllowListedTransformer = func() []string {
		transformers := make([]string, 0, len(transformerAllowlist))
		for funcName := range transformerAllowlist {
			transformers = append(transformers, funcName)
		}
		return transformers
	}()
)

// NewTransformer instantiates a Transformer object.
func NewTransformer(config []latestV2.Transformer) (*Transformer, error) {
	newFuncs, err := validateTransformers(config)
	if err != nil {
		return nil, err
	}
	return &Transformer{kptFn: newFuncs, needRefresh: true, config: config}, nil
}

type Transformer struct {
	needRefresh bool
	kptFn       []kptfile.Function
	config      []latestV2.Transformer
}

// GetDeclarativeValidators transforms and returns the skaffold validators defined in skaffold.yaml
func (v *Transformer) GetDeclarativeTransformers() ([]kptfile.Function, error) {
	// TODO: guarantee the v.kptFn is updated once users changed skaffold.yaml file.
	if v.needRefresh {
		newFuncs, err := validateTransformers(v.config)
		if err != nil {
			return nil, err
		}
		v.kptFn = newFuncs
		v.needRefresh = false
	}
	return v.kptFn, nil
}

func validateTransformers(config []latestV2.Transformer) ([]kptfile.Function, error) {
	var newFuncs []kptfile.Function
	for _, c := range config {
		newFunc, ok := transformerAllowlist[c.Name]
		if !ok {
			// TODO: Add links to explain "skaffold-managed mode" and "kpt-managed mode".
			return nil, sErrors.NewErrorWithStatusCode(
				&proto.ActionableErr{
					Message: fmt.Sprintf("unsupported transformer %q", c.Name),
					ErrCode: proto.StatusCode_CONFIG_UNKNOWN_TRANSFORMER,
					Suggestions: []*proto.Suggestion{
						{
							SuggestionCode: proto.SuggestionCode_CONFIG_ALLOWLIST_transformers,
							Action: fmt.Sprintf(
								"please only use the following transformers in skaffold-managed mode: %v. "+
									"to use custom transformers, please use kpt-managed mode.", allowListedTransformer),
						},
					},
				})
		}
		if c.ConfigMap != nil {
			for _, stringifiedData := range c.ConfigMap {
				items := strings.Split(stringifiedData, ":")
				if len(items) != 2 {
					return nil, sErrors.NewErrorWithStatusCode(
						&proto.ActionableErr{
							Message: fmt.Sprintf("unknown arguments for transformer %v", c.Name),
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
				newFunc.ConfigMap[items[0]] = items[1]
			}
		}
		newFuncs = append(newFuncs, newFunc)
	}
	return newFuncs, nil
}
