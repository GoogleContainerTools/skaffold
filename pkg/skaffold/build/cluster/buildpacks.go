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

package cluster

import (
	"context"
	"io"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func (b *Builder) buildWithBuildpacks(ctx context.Context, out io.Writer, workspace string, artifact *latest.BuildpackArtifact, tag string) (string, error) {
	client, err := kubernetes.Client()
	if err != nil {
		return "", errors.Wrap(err, "getting Kubernetes client")
	}
	pods := client.CoreV1().Pods(b.Namespace)

	podSpec, err := b.buildpacksPodSpec(artifact, tag)
	if err != nil {
		return "", err
	}

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

	if err := b.copyBuildpacksBuildContext(ctx, workspace, artifact, pods, pod.Name); err != nil {
		return "", errors.Wrap(err, "copying sources")
	}

	// Wait for the pods to succeed while streaming the logs
	waitForLogs := streamLogs(ctx, out, pod.Name, pods)

	if err := kubernetes.WaitForPodSucceeded(ctx, pods, pod.Name, b.timeout); err != nil {
		waitForLogs()
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	waitForLogs()

	return docker.RemoteDigest(tag, b.insecureRegistries)
}

// first copy over the buildcontext tarball into the init container tmp dir via kubectl cp
// Via kubectl exec, we extract the tarball to the empty dir
// Then, via kubectl exec, create the /tmp/complete file via kubectl exec to complete the init container
func (b *Builder) copyBuildpacksBuildContext(ctx context.Context, workspace string, artifact *latest.BuildpackArtifact, pods corev1.PodInterface, podName string) error {
	if err := kubernetes.WaitForPodInitialized(ctx, pods, podName); err != nil {
		return errors.Wrap(err, "waiting for pod to initialize")
	}

	deps, err := buildpacks.GetDependencies(ctx, workspace, artifact)
	if err != nil {
		return err
	}

	var paths []string
	for _, dep := range deps {
		paths = append(paths, filepath.Join(workspace, dep))
	}

	uid := 1000
	gid := 1000
	modTime := time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)

	r, w := io.Pipe()
	go func() {
		if err := util.CreateTarWithParents(w, workspace, paths, uid, gid, modTime); err != nil {
			w.CloseWithError(err)
		} else {
			w.Close()
		}
	}()

	if err := b.kubectlcli.Run(ctx, r, nil, "exec", "-i", podName, "-c", "upload", "-n", b.Namespace, "--", "tar", "-xf", "-", "-C", "/workspace"); err != nil {
		return errors.Wrap(err, "uploading build context")
	}

	// Generate a file to successfully terminate the init container
	return b.kubectlcli.Run(ctx, nil, nil, "exec", podName, "-c", "upload", "-n", b.Namespace, "--", "touch", "/tmp/complete")
}
