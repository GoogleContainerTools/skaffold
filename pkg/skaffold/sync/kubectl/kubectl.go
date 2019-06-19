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

package kubectl

import (
	"context"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type Syncer struct {
	namespaces []string
}

func NewSyncer(namespaces []string) *Syncer {
	return &Syncer{
		namespaces: namespaces,
	}
}

func (k *Syncer) Sync(ctx context.Context, s *sync.Item) error {
	if len(s.Copy) > 0 {
		logrus.Infoln("Copying files:", s.Copy, "to", s.Image)

		if err := sync.Perform(ctx, s.Image, s.Copy, copyFileFn, k.namespaces); err != nil {
			return errors.Wrap(err, "copying files")
		}
	}

	if len(s.Delete) > 0 {
		logrus.Infoln("Deleting files:", s.Delete, "from", s.Image)

		if err := sync.Perform(ctx, s.Image, s.Delete, deleteFileFn, k.namespaces); err != nil {
			return errors.Wrap(err, "deleting files")
		}
	}

	return nil
}

func deleteFileFn(ctx context.Context, pod v1.Pod, container v1.Container, files map[string][]string) []*exec.Cmd {
	// "kubectl" is below...
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
