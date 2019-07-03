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
)

func TestPortForwardEntryKey(t *testing.T) {
	tests := []struct {
		description string
		pfe         *portForwardEntry
		expected    string
	}{
		{
			description: "entry for pod",
			pfe: &portForwardEntry{
				resource: latest.PortForwardResource{
					Type:      "pod",
					Name:      "podName",
					Namespace: "default",
					Port:      8080,
				},
			},
			expected: "pod-podName-default-8080",
		}, {
			description: "entry for deploy",
			pfe: &portForwardEntry{
				resource: latest.PortForwardResource{
					Type:      "deployment",
					Name:      "depName",
					Namespace: "namespace",
					Port:      9000,
				},
			},
			expected: "deployment-depName-namespace-9000",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			acutalKey := test.pfe.key()

			if acutalKey != test.expected {
				t.Fatalf("port forward entry key is incorrect: \n actual: %s \n expected: %s", acutalKey, test.expected)
			}

			if test.pfe.String() != test.expected {
				t.Fatalf("port forward entry string is incorrect: \n actual: %s \n expected: %s", acutalKey, test.expected)
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
			pfe: &portForwardEntry{
				containerName: "containerName",
				portName:      "portName",
				resource: latest.PortForwardResource{
					Type:      "pod",
					Name:      "podName",
					Namespace: "default",
					Port:      8080,
				},
				automaticPodForwarding: true,
			},
			expected: "containerName-default-portName-8080",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			acutalKey := test.pfe.key()

			if acutalKey != test.expected {
				t.Fatalf("port forward entry key is incorrect: \n actual: %s \n expected: %s", acutalKey, test.expected)
			}

			if strings.Contains(acutalKey, "pod") {
				t.Fatal("key should not contain podname, otherwise containers will be mapped to a new port every time a pod is regenerated. See Issues #1815 and #1594.")
			}
		})
	}
}
