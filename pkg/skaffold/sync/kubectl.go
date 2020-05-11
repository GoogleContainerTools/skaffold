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

package sync

import (
	"context"
	"io"
	"os/exec"

	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func (s *podSyncer) deleteFileFn(ctx context.Context, pod v1.Pod, container v1.Container, files syncMap) *exec.Cmd {
	args := make([]string, 0, 9+len(files))
	args = append(args, pod.Name, "--namespace", pod.Namespace, "-c", container.Name, "--", "rm", "-rf", "--")
	for _, dsts := range files {
		args = append(args, dsts...)
	}
	return s.kubectl.Command(ctx, "exec", args...)
}

func (s *podSyncer) copyFileFn(ctx context.Context, pod v1.Pod, container v1.Container, files syncMap) *exec.Cmd {
	// Use "m" flag to touch the files as they are copied.
	reader, writer := io.Pipe()
	go func() {
		if err := util.CreateMappedTar(writer, "/", files); err != nil {
			writer.CloseWithError(err)
		} else {
			writer.Close()
		}
	}()

	copyCmd := s.kubectl.Command(ctx, "exec", pod.Name, "--namespace", pod.Namespace, "-c", container.Name, "-i", "--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	copyCmd.Stdin = reader
	return copyCmd
}
