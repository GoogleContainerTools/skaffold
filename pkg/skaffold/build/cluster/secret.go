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
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func (b *Builder) setupPullSecret(out io.Writer) (func(), error) {
	if b.PullSecret == "" && b.PullSecretName == "" {
		return func() {}, nil
	}
	return reCreateSecret(out, b.Namespace, b.PullSecretName, b.PullSecret, constants.DefaultKanikoSecretName)
}

func (b *Builder) setupDockerConfigSecret(out io.Writer) (func(), error) {
	if b.DockerConfig == nil {
		return func() {}, nil
	}
	return reCreateSecret(out, b.Namespace, b.DockerConfig.SecretName, b.DockerConfig.Path, "config.json")
}

func reCreateSecret(out io.Writer, ns string, secretName string, secretPath string, secretkey string) (func(), error) {

	client, err := kubernetes.Client()
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	secrets := client.CoreV1().Secrets(ns)

	if err := deleteSecret(secrets, secretName, secretPath); err != nil {
		return nil, err
	}

	color.Default.Fprintf(out, "Creating secret [%s]...\n", secretName)
	secretData, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading secret")
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
		},
		Data: map[string][]byte{
			secretkey: secretData,
		},
	}

	if _, err := secrets.Create(secret); err != nil {
		return nil, errors.Wrapf(err, "creating secret: %s", err)
	}

	return func() {
		if err := secrets.Delete(secretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("error deleting secret %s, %s", secretName, err)
		}
	}, nil
}

func deleteSecret(secrets corev1.SecretInterface, secretName string, secretPath string) error {

	if secretPath == "" {
		logrus.Debug("No secret specified. Checking for one in the cluster.")

		if _, err := secrets.Get(secretName, metav1.GetOptions{}); err != nil {
			return errors.Wrap(err, "checking for existing secret")
		}
		return nil
	}

	if _, err := secrets.Get(secretName, metav1.GetOptions{}); err == nil {
		logrus.Infof("Deleting existing %s secret", secretName)
		if err := secrets.Delete(secretName, &metav1.DeleteOptions{}); err != nil {
			return errors.Wrapf(err, "error deleting secret %s, %s", secretName, err)
		}
	}
	return nil
}
