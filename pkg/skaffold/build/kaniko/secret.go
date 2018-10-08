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
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *Builder) setupSecret() (func(), error) {
	logrus.Debug("Creating kaniko secret")

	client, err := kubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	secrets := client.CoreV1().Secrets(b.Namespace)

	if b.PullSecret == "" {
		logrus.Debug("No pull secret specified. Checking for one in the cluster.")

		if _, err := secrets.Get(b.PullSecretName, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrap(err, "checking for existing kaniko secret")
		}

		return func() {}, nil
	}

	secretData, err := ioutil.ReadFile(b.PullSecret)
	if err != nil {
		return nil, errors.Wrap(err, "reading secret")
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
		return nil, errors.Wrapf(err, "creating secret: %s", err)
	}

	return func() {
		if err := secrets.Delete(b.PullSecretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting secret")
		}
	}, nil
}
