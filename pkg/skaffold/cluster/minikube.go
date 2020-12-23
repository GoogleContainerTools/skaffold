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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var GetClient = getClient

// To override during tests
var (
	minikubeBinaryFunc = minikubeBinary
	getClusterInfo     = context.GetClusterInfo
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
	if _, err := minikubeBinaryFunc(); err != nil {
		logrus.Tracef("Minikube cluster not detected: %v", err)
		return false
	}
	// short circuit if context is 'minikube'
	if kubeContext == constants.DefaultMinikubeContext {
		return true
	}

	cluster, err := getClusterInfo(kubeContext)
	if err != nil {
		logrus.Tracef("failed to get cluster info: %v", err)
		return false
	}
	if matchClusterCertPath(cluster.CertificateAuthority) {
		logrus.Debugf("Minikube cluster detected: cluster certificate for context %q found inside the minikube directory", kubeContext)
		return true
	}

	if ok, err := matchServerURL(cluster.Server); err != nil {
		logrus.Tracef("failed to match server url: %v", err)
	} else if ok {
		logrus.Debugf("Minikube cluster detected: server url for context %q matches minikube node ip", kubeContext)
		return true
	}
	logrus.Tracef("Minikube cluster not detected for context %q", kubeContext)
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
	filename, err := exec.LookPath("minikube")
	if err != nil {
		return "", errors.New("unable to find minikube executable. Please add it to PATH environment variable")
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", fmt.Errorf("unable to find minikube executable. File not found %s", filename)
	}
	return filename, nil
}

// matchClusterCertPath checks if the cluster certificate for this context is from inside the minikube directory
func matchClusterCertPath(certPath string) bool {
	return certPath != "" && util.IsSubPath(minikubePath(), certPath)
}

// matchServerURL checks if the k8s server url is same as any of the minikube nodes IPs
func matchServerURL(server string) (bool, error) {
	cmd, _ := minikubeExec("profile", "list", "-o", "json")
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return false, fmt.Errorf("getting minikube profiles: %w", err)
	}

	var data profileList
	if err = json.Unmarshal(out, &data); err != nil {
		return false, fmt.Errorf("failed to unmarshal minikube profile list: %w", err)
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		logrus.Tracef("invalid server url: %v", err)
	}

	for _, v := range data.Valid {
		for _, n := range v.Config.Nodes {
			if err == nil && serverURL.Host == fmt.Sprintf("%s:%d", n.IP, n.Port) {
				// TODO: Revisit once https://github.com/kubernetes/minikube/issues/6642 is fixed
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

type profileList struct {
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
