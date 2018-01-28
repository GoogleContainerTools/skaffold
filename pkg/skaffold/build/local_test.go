/*
Copyright 2018 Google LLC

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

package build

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	testutil "github.com/GoogleCloudPlatform/skaffold/test"
	"github.com/moby/moby/client"
)

func TestLocalRun(t *testing.T) {
	var tests = []struct {
		description string
		config      *config.BuildConfig
		out         io.Writer
		newAPI      func() (client.ImageAPIClient, io.Closer, error)

		expectedBuild *BuildResult
		shouldErr     bool
	}{
		{
			description: "single build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI: testutil.NewFakeImageAPIClientCloser,
			expectedBuild: &BuildResult{
				[]Build{
					{
						ImageName: "test",
						Tag:       "test:imageid",
					},
				},
			},
		},
		{
			description: "error image build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI:    testutil.NewFakeImageAPIClientCloserBuildError,
			shouldErr: true,
		},
		{
			description: "error image tag",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI:    testutil.NewFakeImageAPIClientCloserTagError,
			shouldErr: true,
		},
		{
			description: "error api client",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI:    func() (client.ImageAPIClient, io.Closer, error) { return nil, nil, fmt.Errorf("") },
			shouldErr: true,
		},
		{
			description: "bad writer",
			out:         &testutil.BadWriter{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI:    testutil.NewFakeImageAPIClientCloser,
			shouldErr: true,
		},
		{
			description: "error image list",
			out:         &testutil.BadWriter{},
			config: &config.BuildConfig{
				Artifacts: []config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			newAPI:    testutil.NewFakeImageAPIClientCloserListError,
			shouldErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := &LocalBuilder{
				BuildConfig: test.config,
				newAPI:      test.newAPI,
			}
			res, err := l.Run(test.out)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedBuild, res)
		})
	}
}

func TestNewLocalBuilder(t *testing.T) {
	_, err := NewLocalBuilder(&config.BuildConfig{
		Artifacts: []config.Artifact{
			{
				ImageName: "test",
				Workspace: ".",
			},
		},
	})
	if err != nil {
		t.Errorf("New local builder: %s", err)
	}
}
