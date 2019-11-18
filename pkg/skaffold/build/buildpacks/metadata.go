/*
Copyright 2019 The Skaffold Authors

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

package buildpacks

import (
	"context"
	"encoding/json"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type runImage struct {
	Image string `json:"image"`
}

type stack struct {
	RunImage runImage `json:"runImage"`
}

type metadata struct {
	Stack stack `json:"stack"`
}

// TODO(dgageot): mirrors
func (b *Builder) findRunImage(ctx context.Context, a *latest.BuildpackArtifact, builder string) (string, error) {
	if a.RunImage != "" {
		return a.RunImage, nil
	}

	cfg, err := b.localDocker.ConfigFile(ctx, builder)
	if err != nil {
		return "", err
	}

	var m metadata
	label := cfg.Config.Labels["io.buildpacks.builder.metadata"]
	if err := json.Unmarshal([]byte(label), &m); err != nil {
		return "", err
	}

	return m.Stack.RunImage.Image, nil
}

type buildMetadata struct {
	Bom []bom `json:"bom"`
}

type bom struct {
	Metadata bomMetadata `json:"metadata"`
}

type bomMetadata struct {
	Sync []sync `json:"skaffold.sync"`
}

type sync struct {
	Src   string `json:"src"`
	Dest  string `json:"dest"`
	Strip string `json:"strip"`
}

// $ docker inspect demo/buildpacks | jq -r '.[].Config.Labels["io.buildpacks.build.metadata"] | fromjson.bom[].metadata["skaffold.sync"]'
func SyncRules(labels map[string]string) ([]*latest.SyncRule, error) {
	metadataJSON, present := labels["io.buildpacks.build.metadata"]
	if !present {
		return nil, nil
	}

	m := buildMetadata{}
	if err := json.Unmarshal([]byte(metadataJSON), &m); err != nil {
		return nil, err
	}

	var rules []*latest.SyncRule

	for _, b := range m.Bom {
		for _, sync := range b.Metadata.Sync {
			rules = append(rules, &latest.SyncRule{
				Src:   sync.Src,
				Dest:  sync.Dest,
				Strip: sync.Strip,
			})
		}
	}

	return rules, nil
}
