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
	"fmt"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestClientImpl_IsMinikube(t *testing.T) {
	home := homedir.HomeDir()
	tests := []struct {
		description        string
		kubeContext        string
		clusterInfo        clientcmdapi.Cluster
		config             rest.Config
		minikubeProfileCmd util.Command
		minikubeNotInPath  bool
		expected           bool
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
			clusterInfo: clientcmdapi.Cluster{
				CertificateAuthority: filepath.Join(home, ".minikube", "ca.crt"),
			},
			expected: true,
		},
		{
			description: "cluster cert outside minikube dir",
			kubeContext: "test-cluster",
			clusterInfo: clientcmdapi.Cluster{
				CertificateAuthority: filepath.Join(home, "foo", "ca.crt"),
			},
			expected: false,
		},
		{
			description: "minikube profile name with docker driver matches kubeContext",
			kubeContext: "test-cluster",
			config: rest.Config{
				Host: "127.0.0.1:32768",
			},
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json", fmt.Sprintf(profileStr, "test-cluster", "docker", "172.17.0.3", 8443)),
			expected:           true,
		},
		{
			description: "minikube profile name with hyperkit driver node ip matches api server url",
			kubeContext: "test-cluster",
			config: rest.Config{
				Host: "192.168.64.10:8443",
			},
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json", fmt.Sprintf(profileStr, "test-cluster", "hyperkit", "192.168.64.10", 8443)),
			expected:           true,
		},
		{
			description: "minikube profile name different from kubeContext",
			kubeContext: "test-cluster",
			config: rest.Config{
				Host: "127.0.0.1:32768",
			},
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json", fmt.Sprintf(profileStr, "test-cluster2", "docker", "172.17.0.3", 8443)),
			expected:           false,
		},
		{
			description: "minikube with hyperkit driver node ip different from api server url",
			kubeContext: "test-cluster",
			config: rest.Config{
				Host: "192.168.64.10:8443",
			},
			minikubeProfileCmd: testutil.CmdRunOut("minikube profile list -o json", fmt.Sprintf(profileStr, "test-cluster", "hyperkit", "192.168.64.11", 8443)),
			expected:           false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.minikubeNotInPath {
				t.Override(&minikubeBinaryFunc, func() (string, error) { return "", fmt.Errorf("minikube not in PATH") })
			} else {
				t.Override(&minikubeBinaryFunc, func() (string, error) { return "minikube", nil })
			}
			t.Override(&util.DefaultExecCommand, test.minikubeProfileCmd)
			t.Override(&getRestClientConfigFunc, func() (*rest.Config, error) { return &test.config, nil })
			t.Override(&getClusterInfo, func(string) (*clientcmdapi.Cluster, error) { return &test.clusterInfo, nil })

			ok := GetClient().IsMinikube(test.kubeContext)
			t.CheckDeepEqual(test.expected, ok)
		})
	}
}

var profileStr = `{"invalid": [],"valid": [{"Name": "minikube","Status": "Stopped","Config": {"Name": "%s","Driver": "%s","Nodes": [{"Name": "","IP": "%s","Port": %d}]}}]}`
