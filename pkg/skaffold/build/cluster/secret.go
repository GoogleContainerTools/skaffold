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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
)

func (b *Builder) setupPullSecret(out io.Writer) (func(), error) {
	if b.PullSecret == "" && b.PullSecretName == "" {
		return func() {}, nil
	}

	color.Default.Fprintf(out, "Creating kaniko secret [%s/%s]...\n", b.Namespace, b.PullSecretName)

	client, err := kubernetes.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	secrets := client.CoreV1().Secrets(b.Namespace)

	if b.PullSecret == "" {
		logrus.Debug("No pull secret specified. Checking for one in the cluster.")

		if _, err := secrets.Get(b.PullSecretName, metav1.GetOptions{}); err != nil {
			return nil, fmt.Errorf("checking for existing kaniko secret: %w", err)
		}

		return func() {}, nil
	}

	secretData, err := ioutil.ReadFile(b.PullSecret)
	if err != nil {
		return nil, fmt.Errorf("reading pull secret: %w", err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   b.PullSecretName,
			Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
		},
		Data: map[string][]byte{
			constants.DefaultKanikoSecretName: secretData,
		},
	}

	if _, err := secrets.Create(secret); err != nil {
		return nil, fmt.Errorf("creating pull secret %q: %w", b.PullSecretName, err)
	}

	return func() {
		if err := secrets.Delete(b.PullSecretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting pull secret")
		}
	}, nil
}

func (b *Builder) setupDockerConfigSecret(out io.Writer) (func(), error) {
	if b.DockerConfig == nil {
		return func() {}, nil
	}

	color.Default.Fprintf(out, "Creating docker config secret [%s]...\n", b.DockerConfig.SecretName)

	client, err := kubernetes.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	secrets := client.CoreV1().Secrets(b.Namespace)

	if b.DockerConfig.Path == "" {
		logrus.Debug("No docker config specified. Checking for one in the cluster.")

		if _, err := secrets.Get(b.DockerConfig.SecretName, metav1.GetOptions{}); err != nil {
			return nil, fmt.Errorf("checking for existing kaniko secret: %w", err)
		}

		return func() {}, nil
	}

	secretData, err := ioutil.ReadFile(b.DockerConfig.Path)
	if err != nil {
		return nil, fmt.Errorf("reading docker config: %w", err)
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   b.DockerConfig.SecretName,
			Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
		},
		Data: map[string][]byte{
			"config.json": secretData,
		},
	}

	if _, err := secrets.Create(secret); err != nil {
		return nil, fmt.Errorf("creating docker config secret %q: %w", b.DockerConfig.SecretName, err)
	}

	return func() {
		if err := secrets.Delete(b.DockerConfig.SecretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting docker config secret")
		}
	}, nil
}
