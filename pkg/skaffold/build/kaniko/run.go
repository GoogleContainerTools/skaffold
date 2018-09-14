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
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	cstorage "cloud.google.com/go/storage"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha3"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const kanikoContainerName = "kaniko"

func runKaniko(ctx context.Context, out io.Writer, artifact *v1alpha3.Artifact, cfg *v1alpha3.KanikoBuild) (string, error) {
	dockerfilePath := artifact.DockerArtifact.DockerfilePath

	initialTag := util.RandomID()
	volumes := []v1.Volume{{
		Name: constants.DefaultKanikoSecretName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: cfg.PullSecretName,
			},
		},
	}}

	volumeMounts := []v1.VolumeMount{{
		Name:      constants.DefaultKanikoSecretName,
		MountPath: "/secret",
	}}

	context := ""
	if cfg.BuildContext.GCSBucket != "" {
		tarName := fmt.Sprintf("context-%s.tar.gz", initialTag)
		if err := docker.UploadContextToGCS(ctx, artifact.Workspace, artifact.DockerArtifact, cfg.BuildContext.GCSBucket, tarName); err != nil {
			return "", errors.Wrap(err, "uploading tar to gcs")
		}
		defer gcsDelete(ctx, cfg.BuildContext.GCSBucket, tarName)
		context = fmt.Sprintf("gs://%s/%s", cfg.BuildContext.GCSBucket, tarName)
	} else if cfg.BuildContext.LocalDir {
		// Create the config map
		if err := configMapCreate(artifact, cfg.Namespace, initialTag); err != nil {
			return "", errors.Wrap(err, "creating config map")
		}
		defer configMapDelete(initialTag, cfg.Namespace)
		// Add the config map to volumes
		volumes = append(volumes, v1.Volume{
			Name: constants.DefaultKanikoConfigMapName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: configMapName(initialTag),
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      constants.DefaultKanikoConfigMapName,
			MountPath: constants.DefaultKanikoConfigMapMountPath,
		})
		// the configMap stores symlinks to the files at the specified MountPath and stores the
		// actual files themselves at MountPath/..data
		context = fmt.Sprintf("dir://%s", filepath.Join(constants.DefaultKanikoConfigMapMountPath, "..data"))
	}

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

	p, err := pods.Create(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko",
			Labels:       map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			Namespace:    cfg.Namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            kanikoContainerName,
					Image:           constants.DefaultKanikoImage,
					ImagePullPolicy: v1.PullIfNotPresent,
					Args:            args,
					VolumeMounts:    volumeMounts,
					Env: []v1.EnvVar{{
						Name:  "GOOGLE_APPLICATION_CREDENTIALS",
						Value: "/secret/kaniko-secret",
					}},
				},
			},
			Volumes:       volumes,
			RestartPolicy: v1.RestartPolicyNever,
		},
	})
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

func configMapCreate(artifact *v1alpha3.Artifact, ns, tag string) error {
	logrus.Infof("Creating config map %s", configMapName(tag))
	paths, err := docker.GetDependencies(artifact.Workspace, artifact.DockerArtifact)
	if err != nil {
		return errors.Wrap(err, "getting sources for configmap")
	}
	var files []string
	for _, path := range paths {
		files = append(files, []string{"--from-file", path}...)
	}
	cmd := exec.Command("kubectl", append([]string{"create", "configmap", configMapName(tag), "-n", ns}, files...)...)
	return util.RunCmd(cmd)
}

func configMapDelete(tag, ns string) error {
	logrus.Infof("Deleting config map %s", configMapName(tag))
	cmd := exec.Command("kubectl", "delete", "configmap", configMapName(tag), "-n", ns)
	return util.RunCmd(cmd)
}

func configMapName(tag string) string {
	return fmt.Sprintf("%s-%s", constants.DefaultKanikoConfigMapName, tag)
}
