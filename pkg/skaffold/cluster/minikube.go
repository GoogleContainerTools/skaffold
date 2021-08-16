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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/blang/semver"
	"k8s.io/client-go/util/homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kctx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

var (
	GetClient                   = getClient
	minikubeVersionWithUserFlag = semver.MustParse("1.18.0")
)

// To override during tests
var (
	FindMinikubeBinary    = minikubeBinary
	getClusterInfo        = kctx.GetClusterInfo
	GetCurrentVersionFunc = getCurrentVersion

	findOnce sync.Once
	mk       = struct {
		err     error // determines if version and path are valid
		version semver.Version
		path    string
	}{}
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
	if _, _, err := FindMinikubeBinary(); err != nil {
		log.Entry(context.Background()).Tracef("Minikube cluster not detected: %v", err)
		return false
	}
	// short circuit if context is 'minikube'
	if kubeContext == constants.DefaultMinikubeContext {
		return true
	}

	cluster, err := getClusterInfo(kubeContext)
	if err != nil {
		log.Entry(context.Background()).Tracef("failed to get cluster info: %v", err)
		return false
	}
	if matchClusterCertPath(cluster.CertificateAuthority) {
		log.Entry(context.Background()).Debugf("Minikube cluster detected: cluster certificate for context %q found inside the minikube directory", kubeContext)
		return true
	}

	if ok, err := matchServerURL(cluster.Server); err != nil {
		log.Entry(context.Background()).Tracef("failed to match server url: %v", err)
	} else if ok {
		log.Entry(context.Background()).Debugf("Minikube cluster detected: server url for context %q matches minikube node ip", kubeContext)
		return true
	}
	log.Entry(context.Background()).Tracef("Minikube cluster not detected for context %q", kubeContext)
	return false
}

func (clientImpl) MinikubeExec(arg ...string) (*exec.Cmd, error) {
	return minikubeExec(arg...)
}

func minikubeExec(arg ...string) (*exec.Cmd, error) {
	b, v, err := FindMinikubeBinary()
	if err != nil && !errors.As(err, &versionErr{}) {
		return nil, fmt.Errorf("getting minikube executable: %w", err)
	} else if err == nil && supportsUserFlag(v) {
		arg = append(arg, "--user=skaffold")
	}
	return exec.Command(b, arg...), nil
}

func supportsUserFlag(ver semver.Version) bool {
	return ver.GE(minikubeVersionWithUserFlag)
}

// Retrieves minikube version
func getCurrentVersion() (semver.Version, error) {
	cmd := exec.Command("minikube", "version", "--output=json")
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return semver.Version{}, err
	}
	minikubeOutput := map[string]string{}
	err = json.Unmarshal(out, &minikubeOutput)
	if v, ok := minikubeOutput["minikubeVersion"]; ok {
		currentVersion, err := version.ParseVersion(v)
		if err != nil {
			return semver.Version{}, err
		}
		return currentVersion, nil
	}
	return semver.Version{}, err
}

func minikubeBinary() (string, semver.Version, error) {
	findOnce.Do(func() {
		filename, err := exec.LookPath("minikube")
		if err != nil {
			mk.err = errors.New("unable to lookup minikube executable. Please add it to PATH environment variable")
		}
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			mk.err = fmt.Errorf("unable to find minikube executable. File not found %s", filename)
		}
		mk.path = filename
		if v, err := GetCurrentVersionFunc(); err != nil {
			mk.err = versionErr{err: err}
		} else {
			mk.version = v
		}
	})

	return mk.path, mk.version, mk.err
}

type versionErr struct {
	err error
}

func (v versionErr) Error() string {
	return v.err.Error()
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
		log.Entry(context.Background()).Tracef("invalid server url: %v", err)
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
