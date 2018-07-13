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

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func RunKanikoBuild(ctx context.Context, out io.Writer, artifact *v1alpha2.Artifact, cfg *v1alpha2.KanikoBuild) (string, error) {
	dockerfilePath := artifact.DockerArtifact.DockerfilePath

	initialTag := util.RandomID()
	tarName := fmt.Sprintf("context-%s.tar.gz", initialTag)
	if err := docker.UploadContextToGCS(ctx, artifact.Workspace, dockerfilePath, cfg.GCSBucket, tarName); err != nil {
		return "", errors.Wrap(err, "uploading tar to gcs")
	}
	defer gcsDelete(ctx, cfg.GCSBucket, tarName)

	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	imageList := kubernetes.NewImageList()
	imageList.Add(constants.DefaultKanikoImage)

	logger := kubernetes.NewLogAggregator(out, imageList, kubernetes.NewColorPicker([]*v1alpha2.Artifact{artifact}))
	if err := logger.Start(ctx); err != nil {
		return "", errors.Wrap(err, "starting log streamer")
	}

	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	p, err := client.CoreV1().Pods(cfg.Namespace).Create(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    cfg.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "kaniko",
					Image:           constants.DefaultKanikoImage,
					ImagePullPolicy: v1.PullIfNotPresent,
					Args: addBuildArgs([]string{
						fmt.Sprintf("--dockerfile=%s", dockerfilePath),
						fmt.Sprintf("--context=gs://%s/%s", cfg.GCSBucket, tarName),
						fmt.Sprintf("--destination=%s", imageDst),
						fmt.Sprintf("-v=%s", logrus.GetLevel().String()),
					}, artifact),
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      constants.DefaultKanikoSecretName,
							MountPath: "/secret",
						},
					},
					Env: []v1.EnvVar{
						{
							Name:  "GOOGLE_APPLICATION_CREDENTIALS",
							Value: "/secret/kaniko-secret",
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: constants.DefaultKanikoSecretName,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: cfg.PullSecretName,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
	}

	defer func() {
		imageList.Remove(constants.DefaultKanikoImage)
		if err := client.CoreV1().Pods(cfg.Namespace).Delete(p.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	if err := kubernetes.WaitForPodComplete(client.CoreV1().Pods(cfg.Namespace), p.Name); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	return imageDst, nil
}

func gcsDelete(ctx context.Context, bucket, path string) error {
	c, err := cstorage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	return c.Bucket(bucket).Object(path).Delete(ctx)
}

func addBuildArgs(args []string, artifact *v1alpha2.Artifact) []string {
	if artifact.DockerArtifact == nil {
		return args
	}

	if artifact.DockerArtifact.BuildArgs == nil || len(artifact.DockerArtifact.BuildArgs) == 0 {
		return args
	}

	withBuildArgs := make([]string, len(args)+len(artifact.DockerArtifact.BuildArgs))
	copy(withBuildArgs, args)

	i := len(args)
	for k, v := range artifact.DockerArtifact.BuildArgs {
		withBuildArgs[i] = fmt.Sprintf("--build-arg=%s=%s", k, *v)
		i = i + 1
	}

	return withBuildArgs
}
