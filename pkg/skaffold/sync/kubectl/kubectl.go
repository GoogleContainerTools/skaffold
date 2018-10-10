/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

type Syncer struct{}

func (k *Syncer) Sync(ctx context.Context, s *sync.Item) error {
	logrus.Infoln("Copying files:", s.Copy, "to", s.Image)

	if err := sync.Perform(ctx, s.Image, s.Copy, copyFileFn); err != nil {
		return errors.Wrap(err, "copying files")
	}

	logrus.Infoln("Deleting files:", s.Delete, "from", s.Image)

	if err := sync.Perform(ctx, s.Image, s.Delete, deleteFileFn); err != nil {
		return errors.Wrap(err, "deleting files")
	}

	return nil
}

func deleteFileFn(ctx context.Context, pod v1.Pod, container v1.Container, src, dst string) *exec.Cmd {
	return exec.CommandContext(ctx, "kubectl", "exec", pod.Name, "--namespace", pod.Namespace, "-c", container.Name, "--", "rm", "-rf", dst)
}

func copyFileFn(ctx context.Context, pod v1.Pod, container v1.Container, src, dst string) *exec.Cmd {
	return exec.CommandContext(ctx, "kubectl", "cp", src, fmt.Sprintf("%s/%s:%s", pod.Namespace, pod.Name, dst), "-c", container.Name)
}
