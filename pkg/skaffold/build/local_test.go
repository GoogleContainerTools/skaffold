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
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
	"github.com/docker/docker/api/types"
	"github.com/moby/moby/client"
)

type FakeTagger struct {
	Out string
	Err error
}

func (f *FakeTagger) GenerateFullyQualifiedImageName(tagOpts *tag.TagOptions) (string, error) {
	return f.Out, f.Err
}

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) { return nil, nil }
func TestLocalRun(t *testing.T) {
	auth := docker.DefaultAuthHelper
	defer func() { docker.DefaultAuthHelper = auth }()
	docker.DefaultAuthHelper = testAuthHelper{}

	// Set a bad KUBECONFIG path so we don't parse a real one that happens to be
	// present on the host
	unsetEnvs := testutil.SetEnvs(t, map[string]string{"KUBECONFIG": "badpath"})
	defer unsetEnvs(t)
	var tests = []struct {
		description  string
		config       *config.BuildConfig
		out          io.Writer
		newImageAPI  func() (client.ImageAPIClient, io.Closer, error)
		tagger       tag.Tagger
		localCluster bool

		expectedBuild *BuildResult
		shouldErr     bool
	}{
		{
			description: "single build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "gcr.io/test/image",
						Workspace: ".",
					},
				},
				BuildType: config.BuildType{
					LocalBuild: &config.LocalBuild{
						Push: true,
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: testutil.NewFakeImageAPIClientCloser,
			expectedBuild: &BuildResult{
				[]Build{
					{
						ImageName: "gcr.io/test/image",
						Tag:       "gcr.io/test/image:imageid",
					},
				},
			},
		},
		{
			description:  "local cluster bad writer",
			out:          &testutil.BadWriter{},
			shouldErr:    true,
			localCluster: true,
		},
		{
			description: "error image build",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: testutil.NewFakeImageAPIClientCloserBuildError,
			shouldErr:   true,
		},
		{
			description: "error image tag",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: testutil.NewFakeImageAPIClientCloserTagError,
			shouldErr:   true,
		},
		{
			description: "error api client",
			out:         &bytes.Buffer{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: func() (client.ImageAPIClient, io.Closer, error) { return nil, nil, fmt.Errorf("") },
			shouldErr:   true,
		},
		{
			description: "bad writer",
			out:         &testutil.BadWriter{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: testutil.NewFakeImageAPIClientCloser,
			shouldErr:   true,
		},
		{
			description: "error image list",
			out:         &testutil.BadWriter{},
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &tag.ChecksumTagger{},
			newImageAPI: testutil.NewFakeImageAPIClientCloserListError,
			shouldErr:   true,
		},
		{
			description: "error tagger",
			config: &config.BuildConfig{
				Artifacts: []*config.Artifact{
					{
						ImageName: "test",
						Workspace: ".",
					},
				},
			},
			tagger:      &FakeTagger{Err: fmt.Errorf("")},
			newImageAPI: testutil.NewFakeImageAPIClientCloser,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l := &LocalBuilder{
				BuildConfig:  test.config,
				newImageAPI:  test.newImageAPI,
				localCluster: test.localCluster,
			}
			res, err := l.Run(test.out, test.tagger)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedBuild, res)
		})
	}
}

func TestNewLocalBuilder(t *testing.T) {
	_, err := NewLocalBuilder(&config.BuildConfig{
		Artifacts: []*config.Artifact{
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

func TestNewLocalBuilderMinikubeContext(t *testing.T) {
	tmpDir := os.TempDir()
	kubeConfig := filepath.Join(tmpDir, "config")
	defer os.Remove(kubeConfig)
	if err := clientcmd.WriteToFile(api.Config{
		CurrentContext: "minikube",
	}, kubeConfig); err != nil {
		t.Fatalf("writing temp kubeconfig")
	}
	unsetEnvs := testutil.SetEnvs(t, map[string]string{"KUBECONFIG": kubeConfig})
	defer unsetEnvs(t)
	_, err := NewLocalBuilder(&config.BuildConfig{
		Artifacts: []*config.Artifact{
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
