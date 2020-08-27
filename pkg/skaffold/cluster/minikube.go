/*
Copyright 2020 The Skaffold Authors

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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var GetClient = getClient

// To override during tests
var (
	minikubeBinaryFunc      = minikubeBinary
	getRestClientConfigFunc = context.GetRestClientConfig
	getClusterInfo          = context.GetClusterInfo
)

type Client interface {
	// IsMinikube returns true if the given kubeContext maps to a minikube cluster
	IsMinikube(kubeContext string) bool
	// MinikubeExec returns the Cmd struct to execute minikube with given arguments
	MinikubeExec(arg ...string) (*exec.Cmd, error)
}

type clientImpl struct{}

func getClient() Client {
	return clientImpl{}
}

func (clientImpl) IsMinikube(kubeContext string) bool {
	// short circuit if context is 'minikube'
	if kubeContext == constants.DefaultMinikubeContext {
		return true
	}
	_, err := minikubeBinaryFunc()
	if err != nil {
		logrus.Debugf("Minikube cluster not detected: %v", err)
		return false
	}

	if ok, err := matchClusterCertPath(kubeContext); err != nil {
		logrus.Debugf("failed to match cluster certificate path: %v", err)
	} else if ok {
		logrus.Debugf("Minikube cluster detected: cluster certificate for context %q found inside the minikube directory", kubeContext)
		return true
	}

	if ok, err := matchProfileAndServerURL(kubeContext); err != nil {
		logrus.Debugf("failed to match minikube profile: %v", err)
	} else if ok {
		logrus.Debugf("Minikube cluster detected: context %q has matching profile name or server url", kubeContext)
		return true
	}
	logrus.Debugf("Minikube cluster not detected for context %q", kubeContext)
	return false
}

func (clientImpl) MinikubeExec(arg ...string) (*exec.Cmd, error) {
	return minikubeExec(arg...)
}

func minikubeExec(arg ...string) (*exec.Cmd, error) {
	b, err := minikubeBinaryFunc()
	if err != nil {
		return nil, fmt.Errorf("getting minikube executable: %w", err)
	}
	return exec.Command(b, arg...), nil
}

func minikubeBinary() (string, error) {
	execName := "minikube"
	if found, _ := util.DetectWSL(); found {
		execName = "minikube.exe"
	}
	filename, err := exec.LookPath(execName)
	if err != nil {
		return "", errors.New("unable to find minikube executable. Please add it to PATH environment variable")
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", fmt.Errorf("unable to find minikube executable. File not found %s", filename)
	}
	return filename, nil
}

// matchClusterCertPath checks if the cluster certificate for this context is from inside the minikube directory
func matchClusterCertPath(kubeContext string) (bool, error) {
	c, err := getClusterInfo(kubeContext)
	if err != nil {
		return false, fmt.Errorf("getting kubernetes config: %w", err)
	}
	if c.CertificateAuthority == "" {
		return false, nil
	}
	return util.IsSubPath(minikubePath(), c.CertificateAuthority), nil
}

// matchProfileAndServerURL checks if kubecontext matches any valid minikube profile
// and for selected drivers if the k8s server url is same as any of the minikube nodes IPs
func matchProfileAndServerURL(kubeContext string) (bool, error) {
	config, err := getRestClientConfigFunc()
	if err != nil {
		return false, fmt.Errorf("getting kubernetes config: %w", err)
	}
	apiServerURL, _, err := rest.DefaultServerURL(config.Host, config.APIPath, schema.GroupVersion{}, false)

	if err != nil {
		return false, fmt.Errorf("getting kubernetes server url: %w", err)
	}

	logrus.Debugf("kubernetes server url: %s", apiServerURL)

	ok, err := matchServerURLFor(kubeContext, apiServerURL)
	if err != nil {
		return false, fmt.Errorf("checking minikube node url: %w", err)
	}
	return ok, nil
}

func matchServerURLFor(kubeContext string, serverURL *url.URL) (bool, error) {
	cmd, err := minikubeExec("profile", "list", "-o", "json")
	if err != nil {
		return false, fmt.Errorf("executing minikube command: %w", err)
	}

	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return false, fmt.Errorf("getting minikube profiles: %w", err)
	}

	var data data
	if err = json.Unmarshal(out, &data); err != nil {
		log.Fatal(fmt.Errorf("failed to unmarshal data: %w", err))
	}

	for _, v := range data.Valid {
		if v.Config.Name != kubeContext {
			continue
		}

		if v.Config.Driver != "hyperkit" && v.Config.Driver != "virtualbox" {
			// Since node IPs don't match server API for other drivers we assume profile name match is enough.
			// TODO: Revisit once https://github.com/kubernetes/minikube/issues/6642 is fixed
			return true, nil
		}
		for _, n := range v.Config.Nodes {
			if serverURL.Host == fmt.Sprintf("%s:%d", n.IP, n.Port) {
				return true, nil
			}
		}
	}
	return false, nil
}

// minikubePath returns the path to the user's minikube dir
func minikubePath() string {
	minikubeHomeEnv := os.Getenv("MINIKUBE_HOME")
	if minikubeHomeEnv == "" {
		return filepath.Join(homedir.HomeDir(), ".minikube")
	}
	if filepath.Base(minikubeHomeEnv) == ".minikube" {
		return minikubeHomeEnv
	}
	return filepath.Join(minikubeHomeEnv, ".minikube")
}

type data struct {
	Valid   []profile `json:"valid,omitempty"`
	Invalid []profile `json:"invalid,omitempty"`
}

type profile struct {
	Config config
}

type config struct {
	Name   string
	Driver string
	Nodes  []node
}

type node struct {
	IP   string
	Port int32
}
