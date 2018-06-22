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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KanikoBuilder can build docker artifacts on Kubernetes, using Kaniko.
type KanikoBuilder struct {
	*v1alpha2.BuildConfig
}

// NewKanikoBuilder creates a KanikoBuilder.
func NewKanikoBuilder(cfg *v1alpha2.BuildConfig) (*KanikoBuilder, error) {
	return &KanikoBuilder{
		BuildConfig: cfg,
	}, nil
}

// Labels gives labels to be set on artifacts deployed with Kaniko.
func (k *KanikoBuilder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "kaniko",
	}
}

// Build builds a list of artifacts with Kaniko.
func (k *KanikoBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) ([]Artifact, error) {
	teardown, err := k.setupSecret()
	if err != nil {
		return nil, errors.Wrap(err, "setting up secret")
	}
	defer teardown()

	// TODO(r2d4): parallel builds
	var builds []Artifact

	for _, artifact := range artifacts {
		fmt.Fprintf(out, "Building [%s]...\n", artifact.ImageName)

		initialTag, err := kaniko.RunKanikoBuild(ctx, out, artifact, k.KanikoBuild)
		if err != nil {
			return nil, errors.Wrapf(err, "kaniko build for [%s]", artifact.ImageName)
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

func (k *KanikoBuilder) setupSecret() (func(), error) {
	client, err := kubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	secrets := client.CoreV1().Secrets(k.KanikoBuild.Namespace)

	if k.KanikoBuild.PullSecret == "" {
		logrus.Debug("No pull secret specified. Checking for one in the cluster.")

		if _, err := secrets.Get(k.KanikoBuild.PullSecretName, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrap(err, "checking for existing kaniko secret")
		}

		return func() {}, nil
	}

	secretData, err := ioutil.ReadFile(k.KanikoBuild.PullSecret)
	if err != nil {
		return nil, errors.Wrap(err, "reading secret")
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   k.KanikoBuild.PullSecretName,
			Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
		},
		Data: map[string][]byte{
			constants.DefaultKanikoSecretName: secretData,
		},
	}

	if _, err := secrets.Create(secret); err != nil {
		return nil, errors.Wrapf(err, "creating secret: %s", err)
	}

	return func() {
		if err := secrets.Delete(k.KanikoBuild.PullSecretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting secret")
		}
	}, nil
}
