/*
Copyright 2018 Google LLC

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

package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/docker/docker/api"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type DockerAPIClient interface {
	client.CommonAPIClient
	io.Closer
}

// NewDockerAPIClient guesses the docker client to use based on current kubernetes context.
func NewDockerAPIClient(kubeContext string) (DockerAPIClient, error) {
	if kubeContext == constants.DefaultMinikubeContext {
		return NewMinikubeDockerAPIClient()
	}
	return NewEnvDockerAPIClient()
}

// NewEnvDockerAPIClient returns a docker client based on the environment variables set.
// It will "negotiate" the highest possible API version supported by both the client
// and the server if there is a mismatch.
func NewEnvDockerAPIClient() (DockerAPIClient, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Error getting docker client: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	return cli, nil
}

// NewMinikubeDockerAPIClient returns a docker client using the environment variables
// provided by minikube.
func NewMinikubeDockerAPIClient() (DockerAPIClient, error) {
	env, err := getMinikubeDockerEnv()
	if err != nil {
		logrus.Warnf("Could not get minikube docker env, falling back to local docker daemon")
		return NewEnvDockerAPIClient()
	}

	var httpclient *http.Client
	if dockerCertPath := env["DOCKER_CERT_PATH"]; dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env["DOCKER_TLS_VERIFY"] == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		httpclient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
			CheckRedirect: client.CheckRedirect,
		}
	}

	host := env["DOCKER_HOST"]
	if host == "" {
		host = client.DefaultDockerHost
	}
	version := env["DOCKER_API_VERSION"]
	if version == "" {
		version = api.DefaultVersion
	}

	cli, err := client.NewClient(host, version, httpclient, nil)
	if err != nil {
		return cli, err
	}

	return cli, nil
}

func getMinikubeDockerEnv() (map[string]string, error) {
	cmd := exec.Command("minikube", "docker-env", "--shell", "none")
	out, stderr, err := util.RunCommand(cmd, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "getting minikube docker-env stdout: %s, stdin: %s, err: %s", out, stderr, err)
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		kv := strings.Split(line, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Unable to parse minikube docker-env keyvalue: %s, line: %s, output: %s", kv, line, string(out))
		}
		env[kv[0]] = kv[1]
	}
	return env, nil
}
