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
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

const (
	initContainer = "kaniko-init-container"
)

func (b *Builder) buildWithKaniko(ctx context.Context, out io.Writer, workspace string, artifactName string, artifact *latest.KanikoArtifact, tag string, requiredImages map[string]*string, platforms platform.Matcher) (string, error) {
	output.Default.Fprintf(out, "Start building with kaniko for artifact\n")

	start := time.Now()
	defer func() {
		log.Entry(ctx).Infof("Building with kaniko completed in %s", time.Since(start))
	}()

	// TODO: Implement building multi-platform images for cluster builder
	if platforms.IsMultiPlatform() {
		log.Entry(ctx).Warnf("multiple target platforms %q found for artifact %q. Skaffold doesn't yet support multi-platform builds for the docker builder. Consider specifying a single target platform explicitly. See https://skaffold.dev/docs/pipeline-stages/builders/#cross-platform-build-support", platforms.String(), artifactName)
	}

	generatedEnvs, err := generateEnvFromImage(tag)
	if err != nil {
		return "", fmt.Errorf("error processing generated env variables from image uri: %w", err)
	}
	env, err := evaluateEnv(artifact.Env, generatedEnvs...)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate env variables: %w", err)
	}
	artifact.Env = env

	buildArgs, err := docker.EvalBuildArgsWithEnv(b.cfg.Mode(), kaniko.GetContext(artifact, workspace), artifact.DockerfilePath, artifact.BuildArgs, requiredImages, envMapFromVars(artifact.Env))
	if err != nil {
		return "", fmt.Errorf("unable to evaluate build args: %w", err)
	}
	artifact.BuildArgs = buildArgs

	client, err := kubernetesclient.DefaultClient()
	if err != nil {
		return "", fmt.Errorf("getting Kubernetes client: %w", err)
	}
	pods := client.CoreV1().Pods(b.Namespace)

	podSpec, err := b.kanikoPodSpec(artifact, tag, platforms)
	if err != nil {
		return "", err
	}

	pod, err := pods.Create(ctx, podSpec, metav1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("creating kaniko pod: %w", err)
	}

	defer func() {
		// if build interrupted the original context is cancelled
		// and pod deletion will not be called, so we need a new ctx
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := pods.Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			log.Entry(ctx).Errorf("deleting pod: %s", err)
		}
	}()

	if err := b.setupKanikoBuildContext(ctx, out, workspace, artifactName, artifact, pods, pod.Name); err != nil {
		return "", fmt.Errorf("copying sources: %w", err)
	}

	// Wait for the pods to succeed while streaming the logs
	waitForLogs := streamLogs(ctx, out, pod.Name, pods)

	if err := kubernetes.WaitForPodSucceeded(ctx, pods, pod.Name, b.timeout); err != nil {
		waitForLogs()
		return "", err
	}

	waitForLogs()
	if digest := getDigestFromContainerLogs(ctx, pods, pod.Name); digest != "" {
		log.Entry(ctx).Debugf("retrieved image digest %q from kaniko container status message", digest)
		return digest, nil
	}
	log.Entry(ctx).Debug("cannot get image digest from kaniko container status message. Checking directly against the image registry")
	return docker.RemoteDigest(tag, b.cfg, nil)
}

func (b *Builder) copyKanikoBuildContext(ctx context.Context, out io.Writer, workspace string, artifactName string, artifact *latest.KanikoArtifact, podName string) error {
	copyTimeout, err := time.ParseDuration(artifact.CopyTimeout)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, copyTimeout)
	defer cancel()
	errs := make(chan error, 1)
	buildCtxReader, buildCtxWriter := io.Pipe()
	gzipWriter, err := gzip.NewWriterLevel(buildCtxWriter, *artifact.BuildContextCompressionLevel)

	if err != nil {
		return fmt.Errorf("creating gzip writer: %w", err)
	}

	go func() {
		defer func() {
			closeErr := gzipWriter.Close()
			if closeErr != nil {
				log.Entry(ctx).Debugf("closing gzip writer: %v", closeErr)
			}
			closeErr = buildCtxWriter.Close() // it's safe to close the writer multiple times
			if closeErr != nil {
				log.Entry(ctx).Debugf("closing build context writer: %v", closeErr)
			}
		}()

		err := docker.CreateDockerTarContext(ctx, gzipWriter, docker.NewBuildConfig(
			kaniko.GetContext(artifact, workspace), artifactName, artifact.DockerfilePath, artifact.BuildArgs), b.cfg)
		if err != nil {
			closeErr := buildCtxWriter.CloseWithError(fmt.Errorf("creating docker context: %w", err))
			if closeErr != nil {
				log.Entry(ctx).Debugf("closing build context writer: %v", closeErr)
			}
			errs <- err
			return
		}
	}()

	progressOutput := streamformatter.NewProgressOutput(out)
	progressReader := progress.NewProgressReader(buildCtxReader, progressOutput, 0, "", "Sending build context to Kaniko pod")
	// Send context by piping into `tar`.
	// In case of an error, retry and print the command's output. (The `err` itself is useless: exit status 1).
	var cmdOut bytes.Buffer
	if err := b.kubectlcli.Run(ctx,
		progressReader, &cmdOut, "exec", "-i", podName, "-c", initContainer, "-n", b.Namespace, "--", "tar", "-zxf", "-", "-C", kaniko.DefaultEmptyDirMountPath); err != nil {
		errRun := fmt.Errorf("uploading build context: %s", cmdOut.String())
		select {
		case errTar := <-errs:
			if errTar != nil {
				errRun = fmt.Errorf("%v\ntar errors: %w", errRun, errTar)
			}
			return errRun
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

// first copy over the buildcontext tarball into the init container tmp dir via kubectl cp
// Via kubectl exec, we extract the tarball to the empty dir
// Then, via kubectl exec, create the /tmp/complete file via kubectl exec to complete the init container
func (b *Builder) setupKanikoBuildContext(ctx context.Context, out io.Writer, workspace string, artifactName string, artifact *latest.KanikoArtifact, pods corev1.PodInterface, podName string) error {
	if err := kubernetes.WaitForPodInitialized(ctx, pods, podName); err != nil {
		return fmt.Errorf("waiting for pod to initialize: %w", err)
	}
	// Retry uploading the build context in case of an error.
	// total attempts is `uploadMaxRetries + 1`
	attempt := 1
	timeout, err := time.ParseDuration(artifact.CopyTimeout)
	if err != nil {
		return fmt.Errorf("parsing timeout: %w", err)
	}

	err = wait.Poll(time.Second, timeout*time.Duration(*artifact.CopyMaxRetries+1), func() (bool, error) {
		if err := b.copyKanikoBuildContext(ctx, out, workspace, artifactName, artifact, podName); err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				return false, err
			}
			log.Entry(ctx).Warnf("uploading build context failed, retrying (%d/%d): %v", attempt, *artifact.CopyMaxRetries, err)
			if attempt == *artifact.CopyMaxRetries {
				return false, err
			}
			attempt++
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("uploading build context: %w", err)
	}
	// Generate a file to successfully terminate the init container.
	if out, err := b.kubectlcli.RunOut(ctx, "exec", podName, "-c", initContainer, "-n", b.Namespace, "--", "touch", "/tmp/complete"); err != nil {
		return fmt.Errorf("finishing upload of the build context: %s", out)
	}

	return nil
}

func evaluateEnv(env []v1.EnvVar, additional ...v1.EnvVar) ([]v1.EnvVar, error) {
	// Prepare additional envs
	addEnv := make(map[string]string)
	for _, addEnvVar := range additional {
		addEnv[addEnvVar.Name] = addEnvVar.Value
	}

	// Evaluate provided env variables
	var evaluated []v1.EnvVar
	for _, envVar := range env {
		val, err := util.ExpandEnvTemplate(envVar.Value, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to get value for env variable %q: %w", envVar.Name, err)
		}

		evaluated = append(evaluated, v1.EnvVar{Name: envVar.Name, Value: val})

		// Provided env variables have higher priority than additional (generated) ones
		delete(addEnv, envVar.Name)
	}

	// Append additional (generated) env variables
	for name, value := range addEnv {
		if value != "" {
			evaluated = append(evaluated, v1.EnvVar{Name: name, Value: value})
		}
	}

	return evaluated, nil
}

func envMapFromVars(env []v1.EnvVar) map[string]string {
	envMap := make(map[string]string)
	for _, envVar := range env {
		envMap[envVar.Name] = envVar.Value
	}
	return envMap
}

func generateEnvFromImage(imageStr string) ([]v1.EnvVar, error) {
	imgRef, err := docker.ParseReference(imageStr)
	if err != nil {
		return nil, err
	}
	if imgRef.Tag == "" {
		imgRef.Tag = "latest"
	}
	var generatedEnvs []v1.EnvVar
	generatedEnvs = append(generatedEnvs, v1.EnvVar{Name: "IMAGE_REPO", Value: imgRef.Repo})
	generatedEnvs = append(generatedEnvs, v1.EnvVar{Name: "IMAGE_NAME", Value: imgRef.Name})
	generatedEnvs = append(generatedEnvs, v1.EnvVar{Name: "IMAGE_TAG", Value: imgRef.Tag})
	return generatedEnvs, nil
}

// getDigestFromContainerLogs checks the kaniko container terminated status message for the image digest. This gets set with running the kaniko build with flag --digest-file=/dev/termination-log
func getDigestFromContainerLogs(ctx context.Context, pods corev1.PodInterface, podName string) string {
	pod, err := pods.Get(ctx, podName, metav1.GetOptions{})
	if err != nil || pod == nil {
		return ""
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated != nil {
			return status.State.Terminated.Message
		}
	}
	return ""
}
