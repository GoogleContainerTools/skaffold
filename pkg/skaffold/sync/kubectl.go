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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

func deleteFileFn(ctx context.Context, pod v1.Pod, container v1.Container, files map[string][]string) []*exec.Cmd {
	deleteCmd := []string{"exec", pod.Name, "--namespace", pod.Namespace, "-c", container.Name, "--", "rm", "-rf", "--"}
	args := make([]string, 0, len(deleteCmd)+len(files))
	args = append(args, deleteCmd...)
	for _, dsts := range files {
		args = append(args, dsts...)
	}
	delete := exec.CommandContext(ctx, "kubectl", args...)
	return []*exec.Cmd{delete}
}

func copyFileFn(ctx context.Context, pod v1.Pod, container v1.Container, files map[string][]string) []*exec.Cmd {
	// Use "m" flag to touch the files as they are copied.
	reader, writer := io.Pipe()
	copy := exec.CommandContext(ctx, "kubectl", "exec", pod.Name, "--namespace", pod.Namespace, "-c", container.Name, "-i",
		"--", "tar", "xmf", "-", "-C", "/", "--no-same-owner")
	copy.Stdin = reader
	go func() {
		defer writer.Close()

		if err := util.CreateMappedTar(writer, "/", files); err != nil {
			logrus.Errorln("Error creating tar archive:", err)
		}
	}()
	return []*exec.Cmd{copy}
}
