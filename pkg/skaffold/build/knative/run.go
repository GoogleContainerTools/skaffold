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

package knative

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/knative/build/pkg/apis/build/v1alpha1"
	build_v1alpha1 "github.com/knative/build/pkg/client/clientset/versioned/typed/build/v1alpha1"
	"github.com/knative/build/pkg/logs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func runKnative(ctx context.Context, out io.Writer, a *v1alpha2.Artifact, cfg *v1alpha2.KnativeBuild) (string, error) {
	// TODO move this code and sercret creation to Builder.Build
	if err := testBuildCRD(); err != nil {
		logrus.Errorln("Build CRD is not installed")
		logrus.Errorln("Follow the installation guide: https://github.com/knative/build#getting-started")
		logrus.Errorln("Usually, it's as simple as: kubectl create -f https://storage.googleapis.com/build-crd/latest/release.yaml")
		return "", errors.Wrap(err, "Build CRD is not installed")
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return "", errors.Wrap(err, "getting clientConfig")
	}

	deleteSecret, err := setupSecret(clientConfig, cfg)
	if err != nil {
		return "", errors.Wrap(err, "setting up secret")
	}
	defer deleteSecret()

	deleteServiceAccount, err := setupServiceAccount(clientConfig, cfg)
	if err != nil {
		return "", errors.Wrap(err, "setting up service account")
	}
	defer deleteServiceAccount()

	randomID := util.RandomID()
	zipName := fmt.Sprintf("context-%s.tar.gz", randomID)
	// TODO(dgageot): delete sources
	if err := docker.UploadContextToGCS(ctx, a.Workspace, a.DockerArtifact, cfg.GCSBucket, zipName); err != nil {
		return "", errors.Wrap(err, "uploading tar to gcs")
	}

	client, err := build_v1alpha1.NewForConfig(clientConfig)
	if err != nil {
		return "", errors.Wrap(err, "getting build client")
	}

	imageDst := fmt.Sprintf("%s:%s", a.ImageName, randomID)
	build := buildFor(imageDst, zipName, a, cfg)

	builds := client.Builds(cfg.Namespace)
	build, err = builds.Create(build)
	if err != nil {
		return "", errors.Wrap(err, "creating build")
	}
	defer func() {
		if err := builds.Delete(build.Name, &metav1.DeleteOptions{}); err != nil {
			logrus.Debugln("deleting build", err)
		}
	}()

	if err := logs.Tail(ctx, out, build.Name, build.Namespace); err != nil {
		return "", errors.Wrap(err, "streaming logs")
	}

	if err := waitForCompletion(builds, build); err != nil {
		return "", errors.Wrap(err, "waiting for completion")
	}

	return imageDst, nil
}

func waitForCompletion(builds build_v1alpha1.BuildInterface, build *v1alpha1.Build) error {
	for {
		done, err := builds.Get(build.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "getting status")
		}

		switch buildStatus(done) {
		case v1.ConditionFalse:
			return errors.New("build failed")
		case v1.ConditionTrue:
			return nil
		}
	}
}

func buildStatus(build *v1alpha1.Build) v1.ConditionStatus {
	for _, condition := range build.Status.Conditions {
		if condition.Type == v1alpha1.BuildSucceeded {
			return condition.Status
		}
	}

	return v1.ConditionUnknown
}

func buildFor(imageName, tarName string, a *v1alpha2.Artifact, cfg *v1alpha2.KnativeBuild) *v1alpha1.Build {
	return &v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "skaffold-docker-build",
			Namespace:    cfg.Namespace,
		},
		Spec: v1alpha1.BuildSpec{
			ServiceAccountName: cfg.ServiceAccountName,
			Source: &v1alpha1.SourceSpec{
				Custom: &v1.Container{
					Image:   "gcr.io/cloud-builders/gsutil", // TODO(dgageot) way too big
					Command: []string{"sh", "-c"},
					Args:    []string{fmt.Sprintf("gsutil cp gs://%s/%s - | tar xvz", cfg.GCSBucket, tarName)},
				},
			},
			Steps: []v1.Container{
				{
					Name:  "build-and-push",
					Image: constants.DefaultKanikoImage,
					Args: []string{
						"--dockerfile=/workspace/" + a.ArtifactType.DockerArtifact.DockerfilePath,
						"--destination=" + imageName,
					},
				},
			},
		},
	}
}
