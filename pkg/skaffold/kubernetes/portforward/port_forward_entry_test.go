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

package portforward

import (
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPortForwardEntryKey(t *testing.T) {
	tests := []struct {
		description string
		pfe         *portForwardEntry
		expected    string
	}{
		{
			description: "entry for pod",
			pfe: newPortForwardEntry(0, latest.PortForwardResource{
				Type:      "pod",
				Name:      "podName",
				Namespace: "default",
				Port:      8080,
			}, "", "", "", "", 0, false),
			expected: "pod-podName-default-8080",
		}, {
			description: "entry for deploy",
			pfe: newPortForwardEntry(0, latest.PortForwardResource{
				Type:      "deployment",
				Name:      "depName",
				Namespace: "namespace",
				Port:      9000,
			}, "", "", "", "", 0, false),
			expected: "deployment-depName-namespace-9000",
		}, {
			description: "entry for deployment with capital normalization",
			pfe: newPortForwardEntry(0, latest.PortForwardResource{
				Type:      "Deployment",
				Name:      "depName",
				Namespace: "namespace",
				Port:      9000,
			}, "", "", "", "", 0, false),
			expected: "deployment-depName-namespace-9000",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actualKey := test.pfe.key()

			if actualKey != test.expected {
				t.Fatalf("port forward entry key is incorrect: \n actual: %s \n expected: %s", actualKey, test.expected)
			}

			if test.pfe.String() != test.expected {
				t.Fatalf("port forward entry string is incorrect: \n actual: %s \n expected: %s", actualKey, test.expected)
			}
		})
	}
}

func TestAutomaticPodForwardingKey(t *testing.T) {
	tests := []struct {
		description string
		pfe         *portForwardEntry
		expected    string
	}{
		{
			description: "entry for automatically port forwarded pod",
			pfe: newPortForwardEntry(0, latest.PortForwardResource{
				Type:      "pod",
				Name:      "podName",
				Namespace: "default",
				Port:      8080,
			}, "", "containerName", "portName", "owner", 0, true),
			expected: "owner-containerName-default-portName-8080",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			actualKey := test.pfe.key()

			if actualKey != test.expected {
				t.Fatalf("port forward entry key is incorrect: \n actual: %s \n expected: %s", actualKey, test.expected)
			}

			if strings.Contains(actualKey, "pod") {
				t.Fatal("key should not contain podname, otherwise containers will be mapped to a new port every time a pod is regenerated. See Issues #1815 and #1594.")
			}
		})
	}
}
