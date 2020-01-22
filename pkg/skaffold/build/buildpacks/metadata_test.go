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
	"errors"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type fakeDocker struct {
	docker.LocalDaemon

	configFiles map[string]*v1.ConfigFile
}

func (f *fakeDocker) ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error) {
	if configFile, present := f.configFiles[image]; present {
		return configFile, nil
	}
	return nil, errors.New("Not found")
}

func TestFindRunImage(t *testing.T) {
	tests := []struct {
		description      string
		artifact         *latest.BuildpackArtifact
		expectedRunImage string
		expectedError    string
	}{
		{
			description: "user specified run image",
			artifact: &latest.BuildpackArtifact{
				RunImage: "custom-image/run",
			},
			expectedRunImage: "custom-image/run",
		},
		{
			description: "default run image",
			artifact: &latest.BuildpackArtifact{
				Builder: "image/build",
			},
			expectedRunImage: "image/run",
		},
		{
			description: "unable to find image",
			artifact: &latest.BuildpackArtifact{
				Builder: "unknown",
			},
			expectedError: `unable to find image "unknown"`,
		},
		{
			description: "invalid-labels",
			artifact: &latest.BuildpackArtifact{
				Builder: "invalid-labels",
			},
			expectedError: `unable to decode image labels for "invalid-labels"`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			localDocker := &fakeDocker{
				configFiles: map[string]*v1.ConfigFile{
					"image/build": {
						Config: v1.Config{
							Labels: map[string]string{
								"io.buildpacks.builder.metadata": `{"stack":{"runImage":{"image": "image/run"}}}`,
							},
						},
					},
					"invalid-labels": {
						Config: v1.Config{
							Labels: map[string]string{
								"io.buildpacks.builder.metadata": "invalid",
							},
						},
					},
				},
			}

			builder := NewArtifactBuilder(localDocker, false)
			runImage, err := builder.findRunImage(context.Background(), test.artifact)

			if test.expectedError == "" {
				t.CheckNoError(err)
				t.CheckDeepEqual(test.expectedRunImage, runImage)
			} else {
				t.CheckErrorContains(test.expectedError, err)
			}
		})
	}
}
