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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
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
		"ensure-name-substring": {
			Image:     "gcr.io/kpt-fn/ensure-name-substring:v0.2.0",
			ConfigMap: map[string]string{},
		},
		"search-replace": {
			Image:     "gcr.io/kpt-fn/search-replace:v0.2.0",
			ConfigMap: map[string]string{},
		},
		"set-enforcement-action": {
			Image:     "gcr.io/kpt-fn/set-enforcement-action:v0.1.0",
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
func NewTransformer(config []latest.Transformer) (Transformer, error) {
	newFuncs, err := validateTransformers(config)
	if err != nil {
		return Transformer{}, err
	}
	return Transformer{kptFn: newFuncs, needRefresh: true, config: config}, nil
}

type Transformer struct {
	needRefresh bool
	kptFn       []kptfile.Function
	config      []latest.Transformer
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

func (v *Transformer) Append(ts ...latest.Transformer) error {
	kptfns, err := validateTransformers(ts)
	if err != nil {
		return err
	}
	v.config = append(v.config, ts...)
	v.kptFn = append(v.kptFn, kptfns...)
	return nil
}

func (v *Transformer) IsEmpty() bool {
	return v.config == nil || len(v.config) == 0
}

func (v *Transformer) Transform(ctx context.Context, ml manifest.ManifestList) (manifest.ManifestList, error) {
	if v.kptFn == nil {
		return ml, nil
	}
	var err error
	for _, transformer := range v.kptFn {
		slice := util.EnvMapToSlice(transformer.ConfigMap, "=")
		args := []string{"fn", "eval", "-i", transformer.Image, "-o", "unwrap", "-", "--"}
		args = append(args, slice...)
		cmd := exec.CommandContext(ctx, "kpt", args...)
		reader := ml.Reader()
		buffer := &bytes.Buffer{}
		cmd.Stdin = reader
		cmd.Stdout = buffer

		err := cmd.Run()
		if err != nil {
			return ml, err
		}
		ml, err = manifest.Load(buffer)
		if err != nil {
			return ml, err
		}
	}
	return ml, err
}

// TransformPath transform manifests in-place in filepath.
func (v *Transformer) TransformPath(path string) error {
	for _, transformer := range v.kptFn {
		kvs := util.EnvMapToSlice(transformer.ConfigMap, "=")
		args := []string{"fn", "eval", "-i", transformer.Image, path, "--"}
		args = append(args, kvs...)
		command := exec.Command("kpt", args...)
		err := command.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func validateTransformers(config []latest.Transformer) ([]kptfile.Function, error) {
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
				index := strings.Index(stringifiedData, ":")
				if index == -1 {
					return nil, sErrors.NewErrorWithStatusCode(
						&proto.ActionableErr{
							Message: fmt.Sprintf("unknown arguments for transformer %v", c.Name),
							ErrCode: proto.StatusCode_CONFIG_UNKNOWN_TRANSFORMER,
							Suggestions: []*proto.Suggestion{
								{
									SuggestionCode: proto.SuggestionCode_CONFIG_ALLOWLIST_transformers,
									Action: fmt.Sprintf("please check if the .transformer field and " +
										"make sure `configMapData` is a list of data in the form of `${KEY}:${VALUE}`"),
								},
							},
						})
				}
				newFunc.ConfigMap[stringifiedData[0:index]] = stringifiedData[index+1:]
			}
		}
		newFuncs = append(newFuncs, newFunc)
	}
	return newFuncs, nil
}
