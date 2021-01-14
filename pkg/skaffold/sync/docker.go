/*
Copyright 2021
The Skaffold Authors

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

package sync

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
)

type containerSyncer struct {
}

func (s *containerSyncer) Sync(ctx context.Context, item *Item) error {
	if len(item.Copy) > 0 {
		logrus.Infoln("Copying files:", item.Copy, "to", item.Image)
		if _, err := util.RunCmdOut(s.copyFileFn(ctx, item.Artifact, item.Copy)); err != nil {
			return fmt.Errorf("copying files: %w", err)
		}
	}

	if len(item.Delete) > 0 {
		logrus.Infoln("Deleting files:", item.Delete, "from", item.Image)
		if _, err := util.RunCmdOut(s.deleteFileFn(ctx, item.Artifact, item.Copy)); err != nil {
			return fmt.Errorf("deleting files: %w", err)
		}
	}

	return nil
}

func (s *containerSyncer) deleteFileFn(ctx context.Context, containerName string, files syncMap) *exec.Cmd {
	var args []string
	args = append(args, "exec", containerName, "rm", "-rf", "--")
	for _, dsts := range files {
		args = append(args, dsts...)
	}
	return exec.CommandContext(ctx, "docker", args...)
}

func (s *containerSyncer) copyFileFn(ctx context.Context, containerName string, files syncMap) *exec.Cmd {
	// Use "m" flag to touch the files as they are copied.
	reader, writer := io.Pipe()
	go func() {
		if err := util.CreateMappedTar(writer, "/", files); err != nil {
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	copyCmd := exec.CommandContext(ctx, "docker", "exec", "-i", containerName, "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	copyCmd.Stdin = reader
	return copyCmd
}
