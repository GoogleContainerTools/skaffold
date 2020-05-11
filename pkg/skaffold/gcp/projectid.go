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

package gcp

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// ExtractProjectID extracts the GCP projectID from a docker image name
// This only works if the imageName is pushed to gcr.io.
func ExtractProjectID(imageName string) (string, error) {
	ref, err := name.ParseReference(imageName, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parsing image name %q: %w", imageName, err)
	}

	registry := ref.Context().Registry.Name()
	if registry == "gcr.io" || strings.HasSuffix(registry, ".gcr.io") {
		parts := strings.Split(imageName, "/")
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("unable to guess GCP projectID from image name [%s]", imageName)
}
