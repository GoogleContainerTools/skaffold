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
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// TODO(dgageot): reuse service account?
func setupServiceAccount(clientConfig *rest.Config, cfg *v1alpha2.KnativeBuild) (func(), error) {
	core, err := corev1.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.ServiceAccountName,
			Namespace: cfg.Namespace,
		},
		Secrets: []v1.ObjectReference{{
			Name: cfg.SecretName,
		}},
	}

	serviceAccounts := core.ServiceAccounts(cfg.Namespace)
	if _, err := serviceAccounts.Create(sa); err != nil {
		return nil, errors.Wrapf(err, "creating service account: %s", err)
	}

	return func() {
		if err := serviceAccounts.Delete(cfg.ServiceAccountName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting service account: %s", cfg.ServiceAccountName)
		}
	}, nil
}

func setupSecret(clientConfig *rest.Config, cfg *v1alpha2.KnativeBuild) (func(), error) {
	if cfg.Secret == "" {
		return nil, errors.New("no pull secret specified")
	}

	core, err := corev1.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	secretData, err := ioutil.ReadFile(cfg.Secret)
	if err != nil {
		return nil, errors.Wrap(err, "reading secret")
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.SecretName,
			Namespace: cfg.Namespace,
			Annotations: map[string]string{
				"build.knative.dev/docker-0": "https://gcr.io", // TODO
			},
		},
		Type: "kubernetes.io/basic-auth",
		Data: map[string][]byte{
			"username": []byte("_json_key"),
			"password": secretData,
		},
	}

	secrets := core.Secrets(cfg.Namespace)
	if _, err := secrets.Create(secret); err != nil {
		return nil, errors.Wrapf(err, "creating secret: %s", err)
	}

	return func() {
		if err := secrets.Delete(cfg.SecretName, &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting secret: %s", cfg.SecretName)
		}
	}, nil
}
