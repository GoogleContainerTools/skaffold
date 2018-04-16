package kaniko

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

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

func Build(ctx context.Context, out io.Writer, artifact *v1alpha2.Artifact) (string, error) {
	dockerfilePath := artifact.KanikoArtifact.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = constants.DefaultDockerfilePath
	}
	initialTag := util.RandomID()
	tarName := "context.tar.gz" // TODO(r2d4): until this is configurable upstream
	if err := docker.UploadContextToGCS(ctx, dockerfilePath, artifact.Workspace, artifact.KanikoArtifact.GCSBucket, tarName); err != nil {
		return "", errors.Wrap(err, "uploading tar to gcs")
	}
	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	secretData, err := ioutil.ReadFile(artifact.KanikoArtifact.PullSecret)
	if err != nil {
		return "", errors.Wrap(err, "reading secret")
	}

	_, err = client.CoreV1().Secrets("default").Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kaniko-secret",
			Labels: map[string]string{"kaniko": "kaniko"},
		},
		Data: map[string][]byte{
			"kaniko-secret": secretData,
		},
	})
	if err != nil {
		logrus.Warnf("creating secret: %s", err)
	}
	defer func() {
		if err := client.CoreV1().Secrets("default").Delete("kaniko-secret", &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting secret")
		}
	}()

	imageList := kubernetes.NewImageList()
	imageList.AddImage(constants.DefaultKanikoImage)

	logger := kubernetes.NewLogAggregator(out, imageList, kubernetes.NewColorPicker([]*v1alpha2.Artifact{artifact}))
	if err := logger.Start(ctx, client.CoreV1()); err != nil {
		return "", errors.Wrap(err, "starting log streamer")
	}
	imageDst := fmt.Sprintf("%s:%s", artifact.ImageName, initialTag)
	p, err := client.CoreV1().Pods("default").Create(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kaniko",
			Labels: map[string]string{"kaniko": "kaniko"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "kaniko",
					Image:           constants.DefaultKanikoImage,
					ImagePullPolicy: v1.PullIfNotPresent,
					Args: []string{
						fmt.Sprintf("--dockerfile=%s", dockerfilePath),
						fmt.Sprintf("--bucket=%s", artifact.KanikoArtifact.GCSBucket),
						fmt.Sprintf("--destination=%s", imageDst),
						fmt.Sprintf("-v=%s", logrus.GetLevel().String()),
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "kaniko-secret",
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
					Name: "kaniko-secret",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "kaniko-secret",
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
		imageList.RemoveImage(constants.DefaultKanikoImage)
		if err := client.CoreV1().Pods("default").Delete(p.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	if err := kubernetes.WaitForPodComplete(client.CoreV1().Pods("default"), p.Name); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	return imageDst, nil
}
