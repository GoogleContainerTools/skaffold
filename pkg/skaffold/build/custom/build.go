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

package custom

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// Build builds an artifact using a custom script
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latestV1.Artifact, tag string) (string, error) {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"BuildType":   "custom",
		"Context":     instrumentation.PII(artifact.Workspace),
		"Destination": instrumentation.PII(tag),
	})
	if err := b.runBuildScript(ctx, out, artifact, tag); err != nil {
		return "", fmt.Errorf("building custom artifact: %w", err)
	}

	if b.pushImages {
		return docker.RemoteDigest(tag, b.cfg)
	}

	imageID, err := b.localDocker.ImageID(ctx, tag)
	if err != nil {
		return "", err
	}
	if imageID == "" {
		return "", fmt.Errorf("the custom script didn't produce an image with tag [%s]", tag)
	}

	return imageID, nil
}
