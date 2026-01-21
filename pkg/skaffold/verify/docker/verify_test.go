/*
Copyright 2023 The Skaffold Authors

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

package docker

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
	testEvent "github.com/GoogleContainerTools/skaffold/v2/testutil/event"
)

type fakeDockerDaemon struct {
	docker.LocalDaemon

	PulledImages []string
	ImgsInDaemon map[string]string
}

func (fd *fakeDockerDaemon) NetworkCreate(ctx context.Context, name string, labels map[string]string) error {
	return nil
}

func (fd *fakeDockerDaemon) Pull(ctx context.Context, out io.Writer, ref string, platform v1.Platform) error {
	fd.PulledImages = append(fd.PulledImages, ref)
	return nil
}

func (fd *fakeDockerDaemon) ImageID(ctx context.Context, ref string) (string, error) {
	img := fd.ImgsInDaemon[ref]
	return img, nil
}

func (fd *fakeDockerDaemon) Run(ctx context.Context, out io.Writer, opts docker.ContainerCreateOpts) (<-chan container.WaitResponse, <-chan error, string, error) {
	statusCh := make(chan container.WaitResponse)
	go func() {
		statusCh <- container.WaitResponse{Error: nil, StatusCode: 0}
	}()
	errCh := make(<-chan error)
	return statusCh, errCh, "", nil
}

func (fd *fakeDockerDaemon) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return types.ImageInspect{
		Config: &dockerspec.DockerOCIImageConfig{},
	}, []byte{}, nil
}

func (fd *fakeDockerDaemon) ContainerExists(ctx context.Context, name string) bool {
	return false
}

func Test_UseLocalImages(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		testEvent.InitializeState([]latest.Pipeline{{}})
		ctx := context.TODO()
		runCtx := &runcontext.RunContext{}

		fDockerDaemon := &fakeDockerDaemon{
			LocalDaemon: docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil),

			ImgsInDaemon: map[string]string{
				"gcr.io/img1:latest": "id111",
				"gcr.io/img3:latest": "id111",
			},
		}

		t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
			return fDockerDaemon, nil
		})

		testCases := []*latest.VerifyTestCase{
			{
				Name:   "t1",
				Config: latest.VerifyConfig{},
				ExecutionMode: latest.VerifyExecutionModeConfig{
					VerifyExecutionModeType: latest.VerifyExecutionModeType{
						LocalExecutionMode: &latest.LocalVerifier{
							UseLocalImages: true,
						},
					},
				},
				Container: latest.VerifyContainer{
					Name:  "container1",
					Image: "gcr.io/img1:latest",
				},
			},
			{
				Name:   "t2",
				Config: latest.VerifyConfig{},
				ExecutionMode: latest.VerifyExecutionModeConfig{
					VerifyExecutionModeType: latest.VerifyExecutionModeType{
						LocalExecutionMode: &latest.LocalVerifier{
							UseLocalImages: true,
						},
					},
				},
				Container: latest.VerifyContainer{
					Name:  "container2",
					Image: "gcr.io/img2:latest",
				},
			},
			{
				Name:   "t3",
				Config: latest.VerifyConfig{},
				ExecutionMode: latest.VerifyExecutionModeConfig{
					VerifyExecutionModeType: latest.VerifyExecutionModeType{
						LocalExecutionMode: &latest.LocalVerifier{},
					},
				},
				Container: latest.VerifyContainer{
					Name:  "container3",
					Image: "gcr.io/img3:latest",
				},
			},
		}

		verifier, err := NewVerifier(ctx, runCtx, &label.DefaultLabeller{}, testCases, nil, "", nil)
		t.CheckError(false, err)

		err = verifier.Verify(ctx, nil, nil)
		t.CheckError(false, err)

		expectedPullImgs := []string{"gcr.io/img2:latest", "gcr.io/img3:latest"}

		t.CheckDeepEqual(expectedPullImgs, fDockerDaemon.PulledImages)
	})
}

func TestGetContainerName(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		description   string
		imageName     string
		containerName string
		expected      string
	}{
		{
			description:   "container name specified",
			imageName:     "gcr.io/cloud-builders/gcloud",
			containerName: "custom-container",
			expected:      "custom-container",
		},
		{
			description:   "invalid container name specified",
			imageName:     "gcr.io/cloud-builders/gcloud",
			containerName: "gcr.io/cloud-builders/gcloud",
			expected:      "gcloud",
		},
		{
			description:   "container name not specified",
			imageName:     "gcr.io/cloud-builders/gcloud",
			containerName: "",
			expected:      "gcloud",
		},
	}

	fakeDockerDaemon := &fakeDockerDaemon{
		LocalDaemon: docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil),
	}

	verifier := &Verifier{
		client: fakeDockerDaemon,
	}

	for _, test := range tests {
		testutil.Run(
			t, test.description, func(t *testutil.T) {
				actual := verifier.getContainerName(ctx, test.imageName, test.containerName)
				t.CheckDeepEqual(test.expected, actual)
			},
		)
	}
}
