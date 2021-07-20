/*
Copyright 2021 The Skaffold Authors

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

package hooks

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestContainerRun(t *testing.T) {
	tests := []struct {
		description string
		objects     []runtime.Object
		cmd         *testutil.FakeCmd
		hook        v1.ContainerHook
		kubeContext string
		selector    containerSelector
		namespaces  []string
		expected    string
		shouldErr   bool
	}{
		{
			description: "image selector; single matching pod",
			objects: []runtime.Object{
				createPodObject("gcr.io/foo/img1:latest", "pod1", "container1", "np1"),
				createPodObject("gcr.io/foo/img2:latest", "pod2", "container2", "np2"),
			},
			cmd:         testutil.CmdRunWithOutput("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo hook", "hook success"),
			hook:        v1.ContainerHook{Command: []string{"foo", "hook"}},
			kubeContext: "context1",
			selector:    runningImageSelector("gcr.io/foo/img1:latest"),
			namespaces:  []string{"np1", "np2"},
			expected:    "hook success",
		},
		{
			description: "image selector; pod image different",
			objects: []runtime.Object{
				createPodObject("gcr.io/foo/img2:latest", "pod1", "container1", "np1"),
			},
			hook:        v1.ContainerHook{Command: []string{"foo", "hook"}},
			kubeContext: "context1",
			selector:    runningImageSelector("gcr.io/foo/img1:latest"),
			namespaces:  []string{"np1", "np2"},
		},
		{
			description: "image selector; pod namespace different",
			objects: []runtime.Object{
				createPodObject("gcr.io/foo/img1:latest", "pod1", "container1", "np3"),
			},
			hook:        v1.ContainerHook{Command: []string{"foo", "hook"}},
			kubeContext: "context1",
			selector:    runningImageSelector("gcr.io/foo/img1:latest"),
			namespaces:  []string{"np1", "np2"},
		},
		{
			description: "image selector; command error",
			objects: []runtime.Object{
				createPodObject("gcr.io/foo/img1:latest", "pod1", "container1", "np1"),
			},
			cmd:         testutil.CmdRunErr("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo hook", errors.New("error")),
			hook:        v1.ContainerHook{Command: []string{"foo", "hook"}},
			kubeContext: "context1",
			selector:    runningImageSelector("gcr.io/foo/img1:latest"),
			namespaces:  []string{"np1", "np2"},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)
			t.Override(&kubernetesclient.Client, func() (kubernetes.Interface, error) {
				return fakeclient.NewSimpleClientset(test.objects...), nil
			})
			h := containerHook{
				cfg:        test.hook,
				cli:        &kubectl.CLI{KubeContext: test.kubeContext},
				selector:   test.selector,
				namespaces: test.namespaces,
			}
			var output bytes.Buffer

			err := h.run(context.Background(), &output)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, output.String())
		})
	}
}

func createPodObject(image, podName, containerName, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "rs",
					Kind: "ReplicaSet",
				},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
		Spec: corev1.PodSpec{Containers: []corev1.Container{
			{
				Name:  containerName,
				Image: image,
			},
		}},
	}
}
