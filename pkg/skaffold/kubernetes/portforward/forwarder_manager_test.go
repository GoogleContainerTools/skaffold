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
	"context"
	"io/ioutil"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewForwarderManager(t *testing.T) {
	tests := []struct {
		description        string
		fmOptions          string
		expectedForwarders int
	}{
		{
			description:        "nil forwarder manager",
			fmOptions:          "",
			expectedForwarders: 0,
		},
		{
			description:        "basic forwarder manager",
			fmOptions:          "user",
			expectedForwarders: 1,
		},
		{
			description:        "two options forwarder manager",
			fmOptions:          "user,debug",
			expectedForwarders: 2,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			options := config.PortForwardOptions{}
			options.Set(test.fmOptions)
			fm := NewForwarderManager(&kubectl.CLI{},
				&kubernetes.ImageList{},
				"",
				"",
				nil,
				options,
				nil)

			if fm != nil {
				t.CheckDeepEqual(test.expectedForwarders, len(fm.forwarders))
			}
		})
	}
}

func TestForwarderManagerZeroValue(t *testing.T) {
	var m *ForwarderManager

	// Should not raise a nil dereference
	m.Start(context.Background(), ioutil.Discard)
	m.Stop()
}

func TestAllPorts(t *testing.T) {
	ports := []v1.ContainerPort{
		{Name: "dlv", ContainerPort: 56286},
		{Name: "http", ContainerPort: 8080},
	}
	container := v1.Container{Name: "test", Ports: ports}
	pod := v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{container}}}
	testutil.CheckDeepEqual(t, ports, allPorts(&pod, container))
}

func TestDebugPorts(t *testing.T) {
	ports := []v1.ContainerPort{
		{Name: "dlv", ContainerPort: 56268},
		{Name: "http", ContainerPort: 8080},
	}
	container := v1.Container{Name: "test", Ports: ports}
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "name", Annotations: map[string]string{"debug.cloud.google.com/config": `{"test":{"runtime":"foo","ports":{"dlv":56268}}}`}},
		Spec:       v1.PodSpec{Containers: []v1.Container{container}}}
	testutil.CheckDeepEqual(t, []v1.ContainerPort{{Name: "dlv", ContainerPort: 56268}}, debugPorts(&pod, container))
}

func TestAddForwarder(t *testing.T) {
	tests := []struct {
		description        string
		fmOptions          string
		expectedForwarders []int
	}{
		{
			description:        "nil forwarder manager",
			fmOptions:          "",
			expectedForwarders: []int{0, 0},
		},
		{
			description:        "basic forwarder manager",
			fmOptions:          "user",
			expectedForwarders: []int{1, 1},
		},
		{
			description:        "two options forwarder manager",
			fmOptions:          "user,debug",
			expectedForwarders: []int{2, 3},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			options := config.PortForwardOptions{}
			options.Set(test.fmOptions)
			fm := NewForwarderManager(&kubectl.CLI{}, &kubernetes.ImageList{}, "", "", nil, options, nil)

			if fm != nil {
				t.CheckDeepEqual(test.expectedForwarders[0], len(fm.forwarders))
				fm.AddPodForwarder(&kubectl.CLI{}, &kubernetes.ImageList{}, "", options)
				t.CheckDeepEqual(test.expectedForwarders[1], len(fm.forwarders))
			}
		})
	}
}
