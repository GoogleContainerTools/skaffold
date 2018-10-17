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

package kaniko

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko/sources"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *Builder) run(ctx context.Context, out io.Writer, artifact *latest.Artifact, cfg *latest.KanikoBuild) (string, error) {
	initialTag := util.RandomID()

	s := sources.Retrieve(cfg)
	context, err := s.Setup(ctx, out, artifact, initialTag)
	if err != nil {
		return "", errors.Wrap(err, "setting up build context")
	}
	defer s.Cleanup(ctx)

	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	args := []string{
		fmt.Sprintf("--dockerfile=%s", artifact.DockerArtifact.DockerfilePath),
		fmt.Sprintf("--context=%s", context),
		fmt.Sprintf("--destination=%s", imageDst),
		fmt.Sprintf("-v=%s", logLevel().String()),
	}
	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)

	pods := client.CoreV1().Pods(cfg.Namespace)
	p, err := pods.Create(s.Pod(args))
	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
	}
	defer func() {
		if err := pods.Delete(p.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	if err := s.ModifyPod(ctx, p); err != nil {
		return "", errors.Wrap(err, "modifying kaniko pod")
	}

	waitForLogs := streamLogs(out, p.Name, pods)

	if err := kubernetes.WaitForPodComplete(ctx, pods, p.Name, b.timeout); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	waitForLogs()

	return imageDst, nil
}
