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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/cluster"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// minikube 1.13.0 renumbered exit codes
const minikubeDriverConfictExitCode = 51
const minikubeExGuestUnavailable = 89
const oldMinikubeBadUsageExitCode = 64
const minikubeExGuestError = 80

const dockerDriver = "docker"
const noneDriver = "none"

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
	ContainerDebugging() bool
	GlobalConfig() string
	GetKubeContext() string
	MinikubeProfile() string
	GetInsecureRegistries() map[string]bool
	Mode() config.RunMode
}

// NewAPIClientImpl guesses the docker client to use based on current Kubernetes context.
func NewAPIClientImpl(ctx context.Context, cfg Config) (LocalDaemon, error) {
	dockerAPIClientOnce.Do(func() {
		env, apiClient, err := newAPIClient(ctx, cfg.GetKubeContext())
		dockerAPIClient = NewLocalDaemon(apiClient, env, cfg.Prune(), cfg)
		dockerAPIClientErr = err
	})

	return dockerAPIClient, dockerAPIClientErr
}

// newAPIClient guesses the docker client to use based on current Kubernetes context.
func newAPIClient(ctx context.Context, kubecontext string) ([]string, client.CommonAPIClient, error) {
	if cluster.GetClient().IsMinikube(ctx, kubecontext) {
		profile, driver, err := getMinikubeProfile(ctx)
		if err != nil {
			return nil, nil, err
		}
		return newMinikubeAPIClient(ctx, profile, driver)
	}

	return newEnvAPIClient()
}

func getMinikubeProfile(ctx context.Context) (string, string, error) {
	cmd, err := cluster.GetClient().MinikubeExec(ctx, "profile", "list", "-o", "json")
	if err != nil {
		return "", "", fmt.Errorf("getting minikube profile: %w", err)
	}
	out, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", "", fmt.Errorf("getting minikube profile: %w", err)
	}
	var data cluster.ProfileList
	if err = json.Unmarshal(out, &data); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal minikube profile list: %w", err)
	}

	profile := data.Valid[0].Config.Name
	driver := data.Valid[0].Config.Driver
	return profile, driver, nil
}

// newEnvAPIClient returns a docker client based on the environment variables set.
// It will "negotiate" the highest possible API version supported by both the client
// and the server if there is a mismatch.
func newEnvAPIClient() ([]string, client.CommonAPIClient, error) {
	var opts = []client.Opt{client.WithHTTPHeaders(getUserAgentHeader())}
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		helper, err := connhelper.GetConnectionHelper(host)
		if err == nil && helper != nil {
			httpClient := &http.Client{
				Transport: &http.Transport{
					DialContext: helper.Dialer,
				},
			}
			opts = append(opts, client.WithHTTPClient(httpClient), client.WithHost(helper.Host))
		} else {
			opts = append(opts, client.FromEnv)
		}
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting docker client: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	return nil, cli, nil
}

type ExitCoder interface {
	ExitCode() int
}

// newMinikubeAPIClient returns a client using the environment variables
// provided by minikube.
func newMinikubeAPIClient(ctx context.Context, minikubeProfile string, minikubeDriver string) ([]string, client.CommonAPIClient, error) {
	if minikubeDriver == noneDriver {
		return newEnvAPIClient()
	}

	if minikubeDriver == dockerDriver {
		return newMinikubeAPIClientWithDockerDriver(ctx, minikubeProfile)
	}

	cmd, err := cluster.GetClient().MinikubeExec(ctx, "image", "load")
	if err != nil {
		var exitError ExitCoder
		if errors.As(err, &exitError) && (exitError.ExitCode() == minikubeExGuestError) {
			return nil, nil, sErrors.MinikubeImageLoadError(minikubeExGuestError, err)
		}
		return nil, nil, err
	}

	err = util.RunCmd(ctx, cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("loading cluster image: %w", err)
	}
	return nil, nil, nil
}

// newMinikubeAPIClientWithDockerDriver returns a docker client using the environment variables
// provided by minikube.
func newMinikubeAPIClientWithDockerDriver(ctx context.Context, minikubeProfile string) ([]string, client.CommonAPIClient, error) {
	env, err := getMinikubeDockerEnv(ctx, minikubeProfile)
	if err != nil {
		// When minikube uses the infamous `none` driver, `minikube docker-env` will exit with
		// code 51 (>= 1.13.0) or 64 (< 1.13.0).  Note that exit code 51 was unused prior to 1.13.0
		// so it is safe to check here without knowing the minikube version.
		var exitError ExitCoder
		if errors.As(err, &exitError) && (exitError.ExitCode() == minikubeDriverConfictExitCode || exitError.ExitCode() == oldMinikubeBadUsageExitCode || exitError.ExitCode() == minikubeExGuestUnavailable) {
			// Let's ignore the error and fall back to local docker daemon.
			log.Entry(context.TODO()).Warnf("Could not get minikube docker env, falling back to local docker daemon: %s", err)
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
		log.Entry(context.TODO()).Infof("Using minikube docker daemon at %s", host)
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
	log.Entry(context.TODO()).Debugf("setting Docker user agent to %s", userAgent)
	return map[string]string{
		"User-Agent": userAgent,
	}
}

func getMinikubeDockerEnv(ctx context.Context, minikubeProfile string) (map[string]string, error) {
	if minikubeProfile == "" {
		return nil, fmt.Errorf("empty minikube profile")
	}
	cmd, err := cluster.GetClient().MinikubeExec(ctx, "docker-env", "--shell", "none", "-p", minikubeProfile)
	if err != nil {
		return nil, fmt.Errorf("executing minikube command: %w", err)
	}
	out, err := util.RunCmdOut(ctx, cmd)
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
