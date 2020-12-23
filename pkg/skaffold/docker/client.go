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
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// minikube 1.13.0 renumbered exit codes
const minikubeDriverConfictExitCode = 51
const oldMinikubeBadUsageExitCode = 64

// For testing
var (
	NewAPIClient = NewAPIClientImpl
)

var (
	dockerAPIClientOnce sync.Once
	dockerAPIClient     LocalDaemon
	dockerAPIClientErr  error
)

type Config interface {
	Prune() bool
	GetKubeContext() string
	MinikubeProfile() string
	GetInsecureRegistries() map[string]bool
	Mode() config.RunMode
}

// NewAPIClientImpl guesses the docker client to use based on current Kubernetes context.
func NewAPIClientImpl(cfg Config) (LocalDaemon, error) {
	dockerAPIClientOnce.Do(func() {
		env, apiClient, err := newAPIClient(cfg.GetKubeContext(), cfg.MinikubeProfile())
		dockerAPIClient = NewLocalDaemon(apiClient, env, cfg.Prune(), cfg)
		dockerAPIClientErr = err
	})

	return dockerAPIClient, dockerAPIClientErr
}

// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
// remove minikubeProfile from here and instead detect it by matching the
// kubecontext API Server to minikube profiles

// newAPIClient guesses the docker client to use based on current Kubernetes context.
func newAPIClient(kubeContext string, minikubeProfile string) ([]string, client.CommonAPIClient, error) {
	if minikubeProfile != "" { // skip validation if explicitly specifying minikubeProfile.
		return newMinikubeAPIClient(minikubeProfile)
	}
	if cluster.GetClient().IsMinikube(kubeContext) {
		return newMinikubeAPIClient(kubeContext)
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

type ExitCoder interface {
	ExitCode() int
}

// newMinikubeAPIClient returns a docker client using the environment variables
// provided by minikube.
func newMinikubeAPIClient(minikubeProfile string) ([]string, client.CommonAPIClient, error) {
	env, err := getMinikubeDockerEnv(minikubeProfile)
	if err != nil {
		// When minikube uses the infamous `none` driver, `minikube docker-env` will exit with
		// code 51 (>= 1.13.0) or 64 (< 1.13.0).  Note that exit code 51 was unused prior to 1.13.0
		// so it is safe to check here without knowing the minikube version.
		var exitError ExitCoder
		if errors.As(err, &exitError) && (exitError.ExitCode() == minikubeDriverConfictExitCode || exitError.ExitCode() == oldMinikubeBadUsageExitCode) {
			// Let's ignore the error and fall back to local docker daemon.
			logrus.Warnf("Could not get minikube docker env, falling back to local docker daemon: %s", err)
			return newEnvAPIClient()
		}

		return nil, nil, err
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

	if host != client.DefaultDockerHost {
		logrus.Infof("Using minikube docker daemon at %s", host)
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

func getMinikubeDockerEnv(minikubeProfile string) (map[string]string, error) {
	if minikubeProfile == "" {
		return nil, fmt.Errorf("empty minikube profile")
	}
	cmd, err := cluster.GetClient().MinikubeExec("docker-env", "--shell", "none", "-p", minikubeProfile)
	if err != nil {
		return nil, fmt.Errorf("executing minikube command: %w", err)
	}
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, fmt.Errorf("getting minikube env: %w", err)
	}

	env := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("unable to parse minikube docker-env keyvalue: %s, line: %s, output: %s", kv, line, string(out))
		}
		env[kv[0]] = kv[1]
	}

	return env, nil
}
