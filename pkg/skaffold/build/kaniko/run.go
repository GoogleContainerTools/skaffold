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
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko/sources"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func runKaniko(ctx context.Context, out io.Writer, artifact *latest.Artifact, cfg *latest.KanikoBuild) (string, error) {
	initialTag := util.RandomID()
	s, err := sources.Retrieve(cfg)
	if err != nil {
		return "", errors.Wrap(err, "retrieving build context")
	}
	context, err := s.Setup(ctx, artifact, cfg, initialTag)
	if err != nil {
		return "", errors.Wrap(err, "setting up build context")
	}
	defer s.Cleanup(ctx, cfg)
	dockerfilePath := artifact.DockerArtifact.DockerfilePath

	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	pods := client.CoreV1().Pods(cfg.Namespace)

	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	args := []string{
		fmt.Sprintf("--dockerfile=%s", dockerfilePath),
		fmt.Sprintf("--context=%s", context),
		fmt.Sprintf("--destination=%s", imageDst),
		fmt.Sprintf("-v=%s", logrus.GetLevel().String()),
	}
	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)

	p, err := pods.Create(s.Pod(cfg, args))
	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
	}
	if err := s.ModifyPod(p); err != nil {
		return "", errors.Wrap(err, "modifying kaniko pod")
	}
	waitForLogs := streamLogs(out, p.Name, pods)

	defer func() {
		if err := pods.Delete(p.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return "", errors.Wrap(err, "parsing timeout")
	}

	if err := kubernetes.WaitForPodComplete(pods, p.Name, timeout); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	waitForLogs()

	return imageDst, nil
}

func streamLogs(out io.Writer, name string, pods corev1.PodInterface) func() {
	var wg sync.WaitGroup
	wg.Add(1)

	var retry int32 = 1
	go func() {
		defer wg.Done()

		for atomic.LoadInt32(&retry) == 1 {
			r, err := pods.GetLogs(name, &v1.PodLogOptions{
				Follow:    true,
				Container: constants.DefaultKanikoContainerName,
			}).Stream()
			if err == nil {
				io.Copy(out, r)
				return
			}

			logrus.Debugln("unable to get kaniko pod logs:", err)
			time.Sleep(1 * time.Second)
		}
	}()

	return func() {
		atomic.StoreInt32(&retry, 0)
		wg.Wait()
	}
}
