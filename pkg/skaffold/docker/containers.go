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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// ContainerRun runs a list of containers in sequence, stopping on the first error.
// TODO: by properly interleaving calls to the Docker API, we could speed
// things up by roughly 700ms.
func (l *localDaemon) ContainerRun(ctx context.Context, out io.Writer, runs ...ContainerRun) error {
	for _, run := range runs {
		container, err := l.apiClient.ContainerCreate(ctx, &container.Config{
			Image: run.Image,
			Cmd:   run.Command,
			User:  run.User,
			Env:   run.Env,
		}, &container.HostConfig{
			Mounts: run.Mounts,
		}, nil, "")
		if err != nil {
			return err
		}

		if run.BeforeStart != nil {
			run.BeforeStart(ctx, container.ID)
		}

		errRun := l.runAndLog(ctx, out, container.ID)

		// Don't use ctx. It might have been cancelled by Ctrl-C
		if err := l.apiClient.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			return err
		}

		if errRun != nil {
			return errRun
		}
	}

	return nil
}

func (l *localDaemon) runAndLog(ctx context.Context, out io.Writer, containerID string) error {
	if err := l.apiClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	logs, err := l.apiClient.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	defer logs.Close()

	_, err = stdcopy.StdCopy(out, out, logs)
	return err
}

// CopyToContainer copies files to a running container.
func (l *localDaemon) CopyToContainer(ctx context.Context, container string, dest string, root string, paths []string, uid, gid int, modTime time.Time) error {
	r, w := io.Pipe()
	go func() {
		if err := util.CreateTarWithParents(w, root, paths, uid, gid, modTime); err != nil {
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}()

	return l.apiClient.CopyToContainer(ctx, container, dest, r, types.CopyToContainerOptions{})
}

// VolumeRemove removes a volume.
func (l *localDaemon) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return l.apiClient.VolumeRemove(ctx, volumeID, force)
}
