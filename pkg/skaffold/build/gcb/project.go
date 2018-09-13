/*
Copyright 2018 The Skaffold Authors

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

package gcb

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

func (b *Builder) guessProjectID(artifact *v1alpha3.Artifact) (string, error) {
	if b.ProjectID != "" {
		return b.ProjectID, nil
	}

	ref, err := reference.ParseNormalizedNamed(artifact.ImageName)
	if err != nil {
		return "", errors.Wrap(err, "parsing image name for registry")
	}

	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	index := repoInfo.Index
	if !index.Official {
		switch index.Name {
		case "gcr.io", "us.gcr.io", "eu.gcr.io", "asia.gcr.io", "staging-k8s.gcr.io":
			parts := strings.Split(repoInfo.Name.String(), "/")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("unable to guess GCP projectID from image name [%s]", artifact.ImageName)
}
