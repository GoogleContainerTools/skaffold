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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

var (
	transformerAllowlist = map[string]kptfile.Function{
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
			return nil, errors.UnknownTransformerError(c.Name, AllowListedTransformer)
		}
		if c.ConfigMapData != nil {
			for _, stringifiedData := range c.ConfigMapData {
				items := strings.Split(stringifiedData, ":")
				if len(items) != 2 {
					return nil, errors.BadTransformerParamsError(c.Name)
				}
				newFunc.ConfigMap[items[0]] = items[1]
			}
		}
		newFuncs = append(newFuncs, newFunc)
	}
	return newFuncs, nil
}
