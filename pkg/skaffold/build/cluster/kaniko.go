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
	"fmt"
	"io"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cluster/sources"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *Builder) runKanikoBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	// Prepare context
	s := sources.Retrieve(b.kubectlcli, b.ClusterDetails, artifact.KanikoArtifact)
	dependencies, err := b.DependenciesForArtifact(ctx, artifact)
	if err != nil {
		return "", errors.Wrapf(err, "getting dependencies for %s", artifact.ImageName)
	}
	context, err := s.Setup(ctx, out, artifact, util.RandomID(), dependencies)
	if err != nil {
		return "", errors.Wrap(err, "setting up build context")
	}
	defer s.Cleanup(ctx)

	args, err := args(artifact.KanikoArtifact, context, tag)
	if err != nil {
		return "", errors.Wrap(err, "building args list")
	}

	// Create pod
	client, err := kubernetes.Client()
	if err != nil {
		return "", errors.Wrap(err, "getting kubernetes client")
	}

	pods := client.CoreV1().Pods(b.Namespace)
	podSpec := s.Pod(args)
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

	err = kubernetes.WaitForPodSucceeded(ctx, pods, pod.Name, b.timeout)
	waitForLogs()
	if err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	return docker.RemoteDigest(tag, b.insecureRegistries)
}

func args(artifact *latest.KanikoArtifact, context, tag string) ([]string, error) {
	// Create pod spec
	args := []string{
		"--dockerfile", artifact.DockerfilePath,
		"--context", context,
		"--destination", tag,
		"-v", logLevel().String()}

	// TODO: remove since AdditionalFlags will be deprecated (priyawadhwa@)
	if artifact.AdditionalFlags != nil {
		logrus.Warn("The additionalFlags field in kaniko is deprecated, please consult the current schema at skaffold.dev to update your skaffold.yaml.")
		args = append(args, artifact.AdditionalFlags...)
	}

	buildArgs, err := docker.EvaluateBuildArgs(artifact.BuildArgs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to evaluate build args")
	}

	if buildArgs != nil {
		var keys []string
		for k := range buildArgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := buildArgs[k]
			if v == nil {
				args = append(args, "--build-arg", k)
			} else {
				args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, *v))
			}
		}
	}

	if artifact.Target != "" {
		args = append(args, "--target", artifact.Target)
	}

	if artifact.Cache != nil {
		args = append(args, "--cache=true")
		if artifact.Cache.Repo != "" {
			args = append(args, "--cache-repo", artifact.Cache.Repo)
		}
		if artifact.Cache.HostPath != "" {
			args = append(args, "--cache-dir", artifact.Cache.HostPath)
		}
	}

	if artifact.Reproducible {
		args = append(args, "--reproducible")
	}

	return args, nil
}
