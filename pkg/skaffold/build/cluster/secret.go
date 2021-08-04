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
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

const (
	defaultKanikoSecretPath = "kaniko-secret"
)

func (b *Builder) setupPullSecret(ctx context.Context, out io.Writer) (func(), error) {
	if b.PullSecretPath == "" && b.PullSecretName == "" {
		return func() {}, nil
	}

	output.Default.Fprintf(out, "Checking for kaniko secret [%s/%s]...\n", b.Namespace, b.PullSecretName)
	client, err := kubernetesclient.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	secrets := client.CoreV1().Secrets(b.Namespace)
	if _, err := secrets.Get(ctx, b.PullSecretName, metav1.GetOptions{}); err != nil {
		output.Default.Fprintf(out, "Creating kaniko secret [%s/%s]...\n", b.Namespace, b.PullSecretName)
		if b.PullSecretPath == "" {
			return nil, fmt.Errorf("secret %s does not exist. No path specified to create it", b.PullSecretName)
		}
		return b.createSecretFromFile(ctx, secrets)
	}
	if b.PullSecretPath == "" {
		// TODO: Remove the warning when pod health check can display pod failure errors.
		logrus.Warnf("Assuming the secret %s is mounted inside Kaniko pod with the filename %s. If your secret is mounted at different path, please specify using config key `pullSecretPath`.\nSee https://skaffold.dev/docs/references/yaml/#build-cluster-pullSecretPath", b.PullSecretName, defaultKanikoSecretPath)
		b.PullSecretPath = defaultKanikoSecretPath
		return func() {}, nil
	}
	return func() {}, nil
}

func (b *Builder) createSecretFromFile(ctx context.Context, secrets typedV1.SecretInterface) (func(), error) {
	secretData, err := ioutil.ReadFile(b.PullSecretPath)
	if err != nil {
		return nil, fmt.Errorf("cannot create secret %s from path %s. reading pull secret: %w", b.PullSecretName, b.PullSecretPath, err)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   b.PullSecretName,
			Labels: map[string]string{"skaffold-kaniko": "skaffold-kaniko"},
		},
		Data: map[string][]byte{
			kaniko.DefaultSecretName: secretData,
		},
	}
	b.PullSecretPath = kaniko.DefaultSecretName
	if _, err := secrets.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("creating pull secret %q: %w", b.PullSecretName, err)
	}

	return func() {
		if err := secrets.Delete(ctx, b.PullSecretName, metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting pull secret")
		}
	}, nil
}

func (b *Builder) setupDockerConfigSecret(ctx context.Context, out io.Writer) (func(), error) {
	if b.DockerConfig == nil {
		return func() {}, nil
	}

	output.Default.Fprintf(out, "Creating docker config secret [%s]...\n", b.DockerConfig.SecretName)

	client, err := kubernetesclient.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	secrets := client.CoreV1().Secrets(b.Namespace)

	if b.DockerConfig.Path == "" {
		logrus.Debug("No docker config specified. Checking for one in the cluster.")

		if _, err := secrets.Get(ctx, b.DockerConfig.SecretName, metav1.GetOptions{}); err != nil {
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

	if _, err := secrets.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("creating docker config secret %q: %w", b.DockerConfig.SecretName, err)
	}

	return func() {
		if err := secrets.Delete(ctx, b.DockerConfig.SecretName, metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting docker config secret")
		}
	}, nil
}
