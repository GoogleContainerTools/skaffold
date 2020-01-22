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

	"github.com/pkg/errors"

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
func (b *Builder) findRunImage(ctx context.Context, a *latest.BuildpackArtifact) (string, error) {
	if a.RunImage != "" {
		return a.RunImage, nil
	}

	cfg, err := b.localDocker.ConfigFile(ctx, a.Builder)
	if err != nil {
		return "", errors.Wrapf(err, "unable to find image %q", a.Builder)
	}

	var m metadata
	label := cfg.Config.Labels["io.buildpacks.builder.metadata"]
	if err := json.Unmarshal([]byte(label), &m); err != nil {
		return "", errors.Wrapf(err, "unable to decode image labels for %q", a.Builder)
	}

	return m.Stack.RunImage.Image, nil
}
