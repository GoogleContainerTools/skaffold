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

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const kanikoContainerName = "kaniko"

func runKaniko(ctx context.Context, out io.Writer, artifact *v1alpha2.Artifact, cfg *v1alpha2.KanikoBuild) (string, error) {
	initialTag := util.RandomID()
	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	tarName := fmt.Sprintf("context-%s.tar.gz", initialTag)
	buildContext := ""

	if cfg.GcsContext != nil {
		if err := docker.UploadContextToGCS(ctx, artifact.Workspace, artifact.DockerArtifact, cfg.GcsContext.GCSBucket, tarName); err != nil {
			return "", errors.Wrap(err, "uploading tar to gcs")
		}
		defer gcsDelete(ctx, cfg.GcsContext.GCSBucket, tarName)
		buildContext = fmt.Sprintf("gs://%s/%s", cfg.GcsContext.GCSBucket, tarName)
	} else if cfg.S3Context != nil {
		if err := docker.UploadContextToS3(ctx, artifact.Workspace, artifact.DockerArtifact, cfg.S3Context.S3Bucket, tarName, cfg.S3Context.Region); err != nil {
			return "", errors.Wrap(err, "uploading tar to s3")
		}
		buildContext = fmt.Sprintf("s3://%s/%s", cfg.S3Context.S3Bucket, tarName)
	} else {
		buildContext = cfg.LocalDirContext.Path
	}

	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	podConfig := buildPodConfig(*artifact, *cfg, imageDst, initialTag, buildContext)
	pods := client.CoreV1().Pods(cfg.Namespace)
	p, err := pods.Create(podConfig)

	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
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

func buildPodConfig(artifact v1alpha2.Artifact, cfg v1alpha2.KanikoBuild, imageDst string, initialTag string, buildContext string) *v1.Pod {
	containers := []v1.Container{}

	args := []string{
		fmt.Sprintf("--dockerfile=%s", artifact.DockerArtifact.DockerfilePath),
		fmt.Sprintf("--context=%s", buildContext),
		fmt.Sprintf("--destination=%s", imageDst),
	}

	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)

	envVars := getEnvVars(cfg)
	volumes := getVolumes(cfg)
	volumeMounts := getVolumeMounts(cfg)

	container := new(v1.Container)
	container.Name = kanikoContainerName
	container.Image = constants.DefaultKanikoImage
	container.ImagePullPolicy = v1.PullIfNotPresent
	container.Args = args
	container.VolumeMounts = volumeMounts
	container.Env = envVars

	containers = append(containers, *container)

	podConfig := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    cfg.Namespace,
		},
		Spec: v1.PodSpec{
			Containers:    containers,
			Volumes:       volumes,
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	return podConfig
}

func getVolumes(cfg v1alpha2.KanikoBuild) []v1.Volume {
	volumes := []v1.Volume{}
	volume := new(v1.Volume)

	volume.Name = constants.DefaultKanikoSecretName
	defaultSecretVolSource := &v1.SecretVolumeSource{
		SecretName: cfg.PullSecretName}
	volume.VolumeSource = v1.VolumeSource{
		Secret: defaultSecretVolSource}

	volumes = append(volumes, *volume)

	for _, thisVolume := range cfg.Volumes {
		volume.Name = thisVolume.Name

		if thisVolume.HostPath != "" {
			hostPathVolSource := &v1.HostPathVolumeSource{
				Path: thisVolume.HostPath}
			volume.VolumeSource = v1.VolumeSource{
				HostPath: hostPathVolSource}
		} else if thisVolume.Secret != "" {
			secretVolSource := &v1.SecretVolumeSource{
				SecretName: thisVolume.Secret}
			volume.VolumeSource = v1.VolumeSource{
				Secret: secretVolSource}
		}
		volumes = append(volumes, *volume)
	}

	return volumes
}

func getVolumeMounts(cfg v1alpha2.KanikoBuild) []v1.VolumeMount {
	volMounts := []v1.VolumeMount{}

	volMount := new(v1.VolumeMount)
	volMount.Name = constants.DefaultKanikoSecretName
	volMount.MountPath = "/secret"

	volMounts = append(volMounts, *volMount)

	for _, thisMount := range cfg.VolumeMounts {
		volMount = new(v1.VolumeMount)
		volMount.Name = thisMount.Name
		volMount.MountPath = thisMount.MountPath
		volMounts = append(volMounts, *volMount)
	}

	return volMounts
}

func getEnvVars(cfg v1alpha2.KanikoBuild) []v1.EnvVar {
	envVars := []v1.EnvVar{}

	for _, thisVar := range cfg.Env {
		envVar := new(v1.EnvVar)
		envVar.Name = thisVar.Name
		envVar.Value = thisVar.Value
		envVars = append(envVars, *envVar)
	}

	return envVars
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
				Container: kanikoContainerName,
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

func gcsDelete(ctx context.Context, bucket, path string) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	return c.Bucket(bucket).Object(path).Delete(ctx)
}
