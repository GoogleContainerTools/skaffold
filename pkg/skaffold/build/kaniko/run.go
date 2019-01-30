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

func (b *Builder) run(ctx context.Context, out io.Writer, artifact *latest.Artifact) (string, error) {
	if artifact.DockerArtifact == nil {
		return "", errors.New("kaniko builder supports only Docker artifacts")
	}

	initialTag := util.RandomID()
	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)

	// Prepare context
	s := sources.Retrieve(b.KanikoBuild)
	context, err := s.Setup(ctx, out, artifact, initialTag)
	if err != nil {
		return "", errors.Wrap(err, "setting up build context")
	}
	defer s.Cleanup(ctx)

	// Create pod spec
	args := []string{
		"--dockerfile", artifact.DockerArtifact.DockerfilePath,
		"--context", context,
		"--destination", imageDst,
		"-v", logLevel().String()}
	args = append(args, b.AdditionalFlags...)
	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)

	if b.Cache != nil {
		args = append(args, "--cache=true")
		if b.Cache.Repo != "" {
			args = append(args, fmt.Sprintf("--cache-repo=%s", b.Cache.Repo))
		}
	}

	podSpec := s.Pod(args)

	// Create pod
	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	pods := client.CoreV1().Pods(b.Namespace)

	pod, err := pods.Create(podSpec)
	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
	}
	defer func() {
		if err := pods.Delete(pod.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	if err := s.ModifyPod(ctx, pod); err != nil {
		return "", errors.Wrap(err, "modifying kaniko pod")
	}

	waitForLogs := streamLogs(out, pod.Name, pods)

	if err := kubernetes.WaitForPodComplete(ctx, pods, pod.Name, b.timeout); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	waitForLogs()

	return imageDst, nil
}
