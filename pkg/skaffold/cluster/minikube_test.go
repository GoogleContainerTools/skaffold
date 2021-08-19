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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestClientImpl_IsMinikube(t *testing.T) {
	home := homedir.HomeDir()
	tests := []struct {
		description             string
		kubeContext             string
		certPath                string
		serverURL               string
		minikubeProfileCmd      util.Command
		minikubeNotInPath       bool
		expected                bool
		minikuneWithoutUserFalg bool
	}{
		{
			description: "context is 'minikube'",
			kubeContext: "minikube",
			expected:    true,
		},
		{
			description:       "'minikube' binary not found",
			kubeContext:       "test-cluster",
			minikubeNotInPath: true,
			expected:          false,
		},
		{
			description: "cluster cert inside minikube dir",
			kubeContext: "test-cluster",
			certPath:    filepath.Join(home, ".minikube", "ca.crt"),
			expected:    true,
		},
		{
			description:             "minikube without user flag",
			kubeContext:             "test-cluster",
			minikubeProfileCmd:      testutil.CmdRunOut("minikube profile list -o json --user=skaffold", fmt.Sprintf(profileStr, "test-cluster", "docker", "172.17.0.3", 8443)),
			certPath:                filepath.Join(home, "foo", "ca.crt"),
			expected:                false,
			minikuneWithoutUserFalg: true,
		},
		{
			description:        "cluster cert outside minikube dir",
			kubeContext:        "test-cluster",
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json --user=skaffold", fmt.Sprintf(profileStr, "test-cluster", "docker", "172.17.0.3", 8443)),
			certPath:           filepath.Join(home, "foo", "ca.crt"),
			expected:           false,
		},
		{
			description:        "minikube node ip matches api server url",
			kubeContext:        "test-cluster",
			serverURL:          "https://192.168.64.10:8443",
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json --user=skaffold", fmt.Sprintf(profileStr, "test-cluster", "hyperkit", "192.168.64.10", 8443)),
			expected:           true,
		},
		{
			description:        "cannot parse minikube profile list",
			kubeContext:        "test-cluster",
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json --user=skaffold", `Random error`),
			expected:           false,
		},
		{
			description:        "minikube node ip different from api server url",
			kubeContext:        "test-cluster",
			serverURL:          "https://192.168.64.10:8443",
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json --user=skaffold", fmt.Sprintf(profileStr, "test-cluster", "hyperkit", "192.168.64.11", 8443)),
			expected:           false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.minikubeNotInPath {
				ver := semver.Version{}
				t.Override(&FindMinikubeBinary, func(context.Context) (string, semver.Version, error) {
					return "", ver, fmt.Errorf("minikube not in PATH")
				})
			} else {
				if test.minikuneWithoutUserFalg {
					ver := semver.Version{Major: 1, Minor: 17, Patch: 0}
					t.Override(&FindMinikubeBinary, func(context.Context) (string, semver.Version, error) {
						return "", ver, fmt.Errorf("minikube not in PATH")
					})
				} else {
					ver := semver.Version{Major: 1, Minor: 18, Patch: 1}
					t.Override(&FindMinikubeBinary, func(context.Context) (string, semver.Version, error) { return "minikube", ver, nil })
				}
			}
			t.Override(&util.DefaultExecCommand, test.minikubeProfileCmd)
			t.Override(&getClusterInfo, func(string) (*clientcmdapi.Cluster, error) {
				return &clientcmdapi.Cluster{
					Server:               test.serverURL,
					CertificateAuthority: test.certPath,
				}, nil
			})

			ok := GetClient().IsMinikube(context.Background(), test.kubeContext)
			t.CheckDeepEqual(test.expected, ok)
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		description string
		versionBlob string
		expected    semver.Version
		shouldErr   bool
	}{
		{
			description: "minikube version incorrect json",
			versionBlob: `{"commit":"ec61815d60f66a6e4f6353030a40b12362557caa-dirty`,
			shouldErr:   true,
		},
		{
			description: "minikube version without the expected key for version",
			versionBlob: `{"commit":"ec61815d60f66a6e4f6353030a40b12362557caa-dirty","minikubeVersionR":"v1.18.0"}`,
		},
		{
			description: "minikube version output as expected",
			versionBlob: `{"commit":"ec61815d60f66a6e4f6353030a40b12362557caa-dirty","minikubeVersion":"v1.18.0"}`,
			expected:    semver.Version{Major: 1, Minor: 18, Patch: 0},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut("minikube version --output=json",
				test.versionBlob),
			)
			actual, err := getCurrentVersion(context.Background())
			t.CheckErrorAndDeepEqual(test.shouldErr, err, actual, test.expected)
		})
	}
}

func TestMinikubeExec(t *testing.T) {
	tests := []struct {
		description string
		version     semver.Version
		err         error
		hasUserArg  bool
		shouldErr   bool
	}{
		{
			description: "minikube binary error",
			err:         fmt.Errorf("could not find minikube"),
			shouldErr:   true,
		},
		{
			description: "minikube version error",
			err:         versionErr{err: fmt.Errorf("wrong version error")},
		},
		{
			description: "minikube semver version v.18.0 and greater",
			version:     semver.Version{Major: 1, Minor: 18, Patch: 1},
			hasUserArg:  true,
		},
		{
			description: "minikube semver version less than v.18.0",
			version:     semver.Version{Major: 1, Minor: 17, Patch: 1},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&FindMinikubeBinary, func(context.Context) (string, semver.Version, error) {
				return "", test.version, test.err
			})
			actual, err := minikubeExec(context.Background(), "test")
			expected := []string{"", "test"}
			if test.hasUserArg {
				expected = append(expected, "--user=skaffold")
			}
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckDeepEqual(actual.Args, expected)
			}
		})
	}
}

var profileStr = `{"invalid": [],"valid": [{"Name": "minikube","Status": "Stopped","Config": {"Name": "%s","Driver": "%s","Nodes": [{"Name": "","IP": "%s","Port": %d}]}}]}`
