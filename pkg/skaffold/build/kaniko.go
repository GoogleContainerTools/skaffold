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

package build

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KanikoBuilder struct {
	*v1alpha2.BuildConfig
}

func NewKanikoBuilder(cfg *v1alpha2.BuildConfig) (*KanikoBuilder, error) {
	return &KanikoBuilder{
		BuildConfig: cfg,
	}, nil
}

func (k *KanikoBuilder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "kaniko",
	}
}

func (k *KanikoBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]Artifact, error) {
	client, err := kubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	if k.KanikoBuild.PullSecret == "" {
		logrus.Debug("No pull secret specified. Checking for one in the cluster.")
		if _, err := client.CoreV1().Secrets(k.KanikoBuild.Namespace).Get(k.KanikoBuild.PullSecretName, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrap(err, "checking for existing kaniko secret")
		}
	} else {
		secretData, err := ioutil.ReadFile(k.KanikoBuild.PullSecret)
		if err != nil {
			return nil, errors.Wrap(err, "reading secret")
		}

		if _, err := client.CoreV1().Secrets(k.KanikoBuild.Namespace).Create(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   k.KanikoBuild.PullSecretName,
				Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
			},
			Data: map[string][]byte{
				constants.DefaultKanikoSecretName: secretData,
			},
		}); err != nil {
			return nil, errors.Wrapf(err, "creating secret: %s", err)
		}
		defer func() {
			if err := client.CoreV1().Secrets(k.KanikoBuild.Namespace).Delete(k.KanikoBuild.PullSecretName, &metav1.DeleteOptions{}); err != nil {
				logrus.Warnf("deleting secret")
			}
		}()
	}

	// TODO(r2d4): parallel builds
	var builds []Artifact

	for _, artifact := range artifacts {
		initialTag, err := kaniko.RunKanikoBuild(ctx, out, artifact, k.KanikoBuild)
		if err != nil {
			return nil, errors.Wrapf(err, "running kaniko build for %s", artifact.ImageName)
		}

		digest, err := docker.RemoteDigest(initialTag)
		if err != nil {
			return nil, errors.Wrap(err, "getting digest")
		}

		tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.Options{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}

		if err := docker.AddTag(initialTag, tag); err != nil {
			return nil, errors.Wrap(err, "tagging image")
		}

		builds = append(builds, Artifact{
			ImageName: artifact.ImageName,
			Tag:       tag,
		})
	}

	return builds, nil
}
