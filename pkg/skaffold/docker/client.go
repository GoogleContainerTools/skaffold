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

package docker

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// For testing
var (
	NewAPIClient = NewAPIClientImpl
)

var (
	dockerAPIClientOnce sync.Once
	dockerAPIClient     LocalDaemon
	dockerAPIClientErr  error
)

// NewAPIClientImpl guesses the docker client to use based on current kubernetes context.
func NewAPIClientImpl(runCtx *runcontext.RunContext) (LocalDaemon, error) {
	dockerAPIClientOnce.Do(func() {
		env, apiClient, err := newAPIClient(runCtx.KubeContext)
		dockerAPIClient = NewLocalDaemon(apiClient, env, runCtx.Opts.Prune(), runCtx.InsecureRegistries)
		dockerAPIClientErr = err
	})

	return dockerAPIClient, dockerAPIClientErr
}

// newAPIClient guesses the docker client to use based on current kubernetes context.
func newAPIClient(kubeContext string) ([]string, client.CommonAPIClient, error) {
	if kubeContext == constants.DefaultMinikubeContext {
		return newMinikubeAPIClient()
	}
	return newEnvAPIClient()
}

// newEnvAPIClient returns a docker client based on the environment variables set.
// It will "negotiate" the highest possible API version supported by both the client
// and the server if there is a mismatch.
func newEnvAPIClient() ([]string, client.CommonAPIClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithHTTPHeaders(getUserAgentHeader()))
	if err != nil {
		return nil, nil, fmt.Errorf("error getting docker client: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	return nil, cli, nil
}

// newMinikubeAPIClient returns a docker client using the environment variables
// provided by minikube.
func newMinikubeAPIClient() ([]string, client.CommonAPIClient, error) {
	env, err := getMinikubeDockerEnv()
	if err != nil {
		logrus.Warnf("Could not get minikube docker env, falling back to local docker daemon: %s", err)
		return newEnvAPIClient()
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
			return nil, nil, err
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

	api, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithHTTPClient(httpclient),
		client.WithHTTPHeaders(getUserAgentHeader()))
	if err != nil {
		return nil, nil, err
	}

	if api != nil {
		api.NegotiateAPIVersion(context.Background())
	}

	// Keep the minikube environment variables
	var environment []string
	for k, v := range env {
		environment = append(environment, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(environment)

	return environment, api, err
}

func getUserAgentHeader() map[string]string {
	userAgent := fmt.Sprintf("skaffold-%s", version.Get().Version)
	logrus.Debugf("setting Docker user agent to %s", userAgent)
	return map[string]string{
		"User-Agent": userAgent,
	}
}

func detectWsl() (bool, error) {
	if _, err := os.Stat("/proc/version"); err == nil {
		b, err := ioutil.ReadFile("/proc/version")
		if err != nil {
			return false, errors.Wrap(err, "read /proc/version")
		}

		if bytes.Contains(b, []byte("Microsoft")) {
			return true, nil
		}
	}
	return false, nil
}

func getMiniKubeFilename() (string, error) {
	if found, _ := detectWsl(); found {
		filename, err := exec.LookPath("minikube.exe")
		if err != nil {
			return "", errors.New("unable to find minikube.exe. Please add it to PATH environment variable")
		}
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return "", fmt.Errorf("unable to find minikube.exe. File not found %s", filename)
		}
		return filename, nil
	}
	return "minikube", nil
}

func getMinikubeDockerEnv() (map[string]string, error) {
	miniKubeFilename, err := getMiniKubeFilename()
	if err != nil {
		return nil, errors.Wrap(err, "getting minikube filename")
	}

	cmd := exec.Command(miniKubeFilename, "docker-env", "--shell", "none")
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "getting minikube env")
	}

	env := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		kv := strings.Split(line, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("unable to parse minikube docker-env keyvalue: %s, line: %s, output: %s", kv, line, string(out))
		}
		env[kv[0]] = kv[1]
	}

	if found, _ := detectWsl(); found {
		cmd := exec.Command("wslpath", env["DOCKER_CERT_PATH"])
		out, err := util.RunCmdOut(cmd)
		if err == nil {
			env["DOCKER_CERT_PATH"] = strings.TrimRight(string(out), "\n")
		} else {
			return nil, fmt.Errorf("can't run wslpath: %s", err)
		}
	}

	return env, nil
}
