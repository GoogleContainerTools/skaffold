/*
Copyright 2018 The Skaffold Authors

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

package kubernetes

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type TestCmdRecorder struct {
	cmds []string
	err  error
}

func (t *TestCmdRecorder) RunCmd(cmd *exec.Cmd) error {
	if t.err != nil {
		return t.err
	}
	t.cmds = append(t.cmds, strings.Join(cmd.Args, " "))
	return nil
}

func (t *TestCmdRecorder) RunCmdOut(cmd *exec.Cmd) ([]byte, error) {
	return nil, t.RunCmd(cmd)
}

func fakeCmd(p v1.Pod, c v1.Container, src, dst string) *exec.Cmd {
	return exec.Command("copy", src, dst)
}

var pod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:   "podname",
		Labels: constants.Labels.DefaultLabels,
	},
	Status: v1.PodStatus{
		Phase: v1.PodRunning,
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "container_name",
				Image: "gcr.io/k8s-skaffold:123",
			},
		},
	},
}

func TestPerform(t *testing.T) {
	var tests = []struct {
		description string
		image       string
		files       map[string]string
		cmdFn       func(v1.Pod, v1.Container, string, string) *exec.Cmd
		cmdErr      error
		clientErr   error
		expected    []string
		shouldErr   bool
	}{
		{
			description: "no error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			expected:    []string{"copy test.go /test.go"},
		},
		{
			description: "cmd error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			cmdErr:      fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "client error",
			image:       "gcr.io/k8s-skaffold:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
			clientErr:   fmt.Errorf(""),
			shouldErr:   true,
		},
		{
			description: "no copy",
			image:       "gcr.io/different-pod:123",
			files:       map[string]string{"test.go": "/test.go"},
			cmdFn:       fakeCmd,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cmdRecord := &TestCmdRecorder{err: test.cmdErr}
			defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
			util.DefaultExecCommand = cmdRecord

			defer func(c func() (kubernetes.Interface, error)) { Client = c }(GetClientset)
			Client = func() (kubernetes.Interface, error) {
				return fake.NewSimpleClientset(pod), test.clientErr
			}

			util.DefaultExecCommand = cmdRecord
			err := perform(test.image, test.files, test.cmdFn)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cmdRecord.cmds)
		})
	}
}
