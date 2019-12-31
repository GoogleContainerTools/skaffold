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

package docker

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type API interface {
	// Local or Remote
	WorkingDir(ctx context.Context, tagged string) (string, error)

	// Remote Operations
	InsecureRegistries() map[string]bool
	RemoteDigest(identifier string) (string, error)
	AddRemoteTag(src, target string) error
	RetrieveRemoteConfig(identifier string) (*v1.ConfigFile, error)
	PushTar(tarPath, tag string) (string, error)

	// Local Operations
	Close() error
	ExtraEnv() ([]string, error)
	ServerVersion(ctx context.Context) (types.Version, error)
	ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error)
	Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error)
	Push(ctx context.Context, out io.Writer, ref string) (string, error)
	Pull(ctx context.Context, out io.Writer, ref string) error
	Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error)
	Tag(ctx context.Context, image, ref string) error
	TagWithImageID(ctx context.Context, ref string, imageID string) (string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error)
	ImageRemove(ctx context.Context, image string, opts types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	ImageExists(ctx context.Context, ref string) (bool, error)
	Prune(ctx context.Context, out io.Writer, images []string, pruneChildren bool) error
	ContainerRun(ctx context.Context, out io.Writer, runs ...ContainerRun) error
	CopyToContainer(ctx context.Context, container string, dest string, root string, paths []string, uid, gid int, modTime time.Time) error
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
}

type dockerAPI struct {
	getAPIClient       func() ([]string, client.CommonAPIClient, error)
	forceRemove        bool
	insecureRegistries map[string]bool
	imageCache         map[string]*v1.ConfigFile
	imageCacheLock     sync.Mutex
}

func NewAPI(runCtx *runcontext.RunContext) API {
	var dockerAPIClientOnce sync.Once
	var extraEnv []string
	var apiClient client.CommonAPIClient
	var err error

	return &dockerAPI{
		getAPIClient: func() ([]string, client.CommonAPIClient, error) {
			dockerAPIClientOnce.Do(func() {
				extraEnv, apiClient, err = newAPIClient(runCtx.KubeContext)
			})
			return extraEnv, apiClient, err
		},
		forceRemove:        runCtx.Opts.Prune(),
		insecureRegistries: runCtx.InsecureRegistries,
		imageCache:         make(map[string]*v1.ConfigFile),
	}
}

func NewAPIForTests(apiClient client.CommonAPIClient, extraEnv []string, forceRemove bool, insecureRegistries map[string]bool) API {
	return &dockerAPI{
		getAPIClient: func() ([]string, client.CommonAPIClient, error) {
			return extraEnv, apiClient, nil
		},
		forceRemove:        forceRemove,
		insecureRegistries: insecureRegistries,
		imageCache:         make(map[string]*v1.ConfigFile),
	}
}

// PushResult gives the information on an image that has been pushed.
type PushResult struct {
	Digest string
}

// BuildResult gives the information on an image that has been built.
type BuildResult struct {
	ID string
}
