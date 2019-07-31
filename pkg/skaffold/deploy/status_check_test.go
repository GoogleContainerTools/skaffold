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

package deploy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	utilpointer "k8s.io/utils/pointer"
)

func TestGetDeployments(t *testing.T) {
	labeller := NewLabeller("")
	tests := []struct {
		description string
		deps        []*appsv1.Deployment
		deadline    map[string]time.Duration
		expected    map[string]time.Duration
		shouldErr   bool
	}{
		{
			description: "multiple deployments in same namespace",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
							"random":            "foo",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(10)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(20)},
				},
			},
			expected: map[string]time.Duration{"dep1": time.Duration(10) * time.Second, "dep2": time.Duration(20) * time.Second},
		}, {
			description: "multiple deployments in same namespace",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
							"random":            "foo",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(300)},
				},
			},
			expected: map[string]time.Duration{"dep1": time.Duration(200) * time.Second},
		}, {
			description: "multiple deployments with no progress deadline set",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
			},
			expected: map[string]time.Duration{"dep1": time.Duration(100) * time.Second,
				"dep2": time.Duration(200) * time.Second},
		},
		{
			description: "no deployments",
			expected:    map[string]time.Duration{},
		},
		{
			description: "multiple deployments in different namespaces",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test1",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
			},
			expected: map[string]time.Duration{"dep1": time.Duration(100) * time.Second},
		},
		{
			description: "deployment in correct namespace but not deployed by skaffold",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							"some-other-tool": "helm",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
			},
			expected: map[string]time.Duration{},
		},
		{
			description: "deployment in correct namespace deployed by skaffold but previous version",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: "skaffold-0.26.0",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
			},
			expected: map[string]time.Duration{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.deps))
			for i, dep := range test.deps {
				objs[i] = dep
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)
			actual, err := getDeployments(client, "test", labeller, time.Duration(200)*time.Second)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

type MockRolloutStatus struct {
	called    int
	responses []string
	err       error
}

func (m *MockRolloutStatus) Executefunc(context.Context, *kubectl.CLI, string) (string, error) {
	var resp string
	if m.err != nil {
		m.called++
		return "", m.err
	}
	if m.called >= len(m.responses) {
		resp = m.responses[len(m.responses)-1]
	} else {
		resp = m.responses[m.called]
	}
	m.called++
	return resp, m.err
}

func TestPollDeploymentRolloutStatus(t *testing.T) {
	tests := []struct {
		description string
		mock        *MockRolloutStatus
		duration    int
		exactCalls  int
		shouldErr   bool
		timedOut    bool
	}{
		{
			description: "rollout returns success",
			mock: &MockRolloutStatus{
				responses: []string{"dep successfully rolled out"},
			},
			exactCalls: 1,
			duration:   50,
		},
		{
			description: "rollout returns error in the first attempt",
			mock: &MockRolloutStatus{
				err: errors.New("deployment.apps/dep could not be found"),
			},
			shouldErr:  true,
			exactCalls: 1,
			duration:   50,
		},
		{
			description: "rollout returns success before time out",
			mock: &MockRolloutStatus{
				responses: []string{
					"Waiting for rollout to finish: 0 of 1 updated replicas are available...",
					"Waiting for rollout to finish: 0 of 1 updated replicas are available...",
					"deployment.apps/dep successfully rolled out"},
			},
			duration:   80,
			exactCalls: 3,
		},
		{
			description: "rollout returns did not stabilize within the given timeout",
			mock: &MockRolloutStatus{
				responses: []string{
					"Waiting for rollout to finish: 1 of 3 updated replicas are available...",
					"Waiting for rollout to finish: 1 of 3 updated replicas are available...",
					"Waiting for rollout to finish: 2 of 3 updated replicas are available..."},
			},
			duration:  100,
			shouldErr: true,
			timedOut:  true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&executeRolloutStatus, test.mock.Executefunc)
			t.Override(&defaultPollPeriodInMilliseconds, 10)

			actual := &sync.Map{}
			pollDeploymentRolloutStatus(context.Background(), &kubectl.CLI{}, "dep", time.Duration(test.duration)*time.Millisecond, actual)

			if _, ok := actual.Load("dep"); !ok {
				t.Error("expected result for deployment dep. But found none")
			}
			err := getSkaffoldDeployStatus(actual)
			t.CheckError(test.shouldErr, err)
			// Check number of calls only if command did not timeout since there could be n-1 or n or n+1 calls when command timed out
			if !test.timedOut {
				t.CheckDeepEqual(test.exactCalls, test.mock.called)
			}
		})
	}
}

func TestGetDeployStatus(t *testing.T) {
	tests := []struct {
		description    string
		deps           map[string]interface{}
		expectedErrMsg []string
		shouldErr      bool
	}{
		{
			description: "one error",
			deps: map[string]interface{}{
				"dep1": "SUCCESS",
				"dep2": fmt.Errorf("could not return within default timeout"),
			},
			expectedErrMsg: []string{"deployment dep2 failed due to could not return within default timeout"},
			shouldErr:      true,
		},
		{
			description: "no error",
			deps: map[string]interface{}{
				"dep1": "SUCCESS",
				"dep2": "RUNNING",
			},
		},
		{
			description: "multiple errors",
			deps: map[string]interface{}{
				"dep1": "SUCCESS",
				"dep2": fmt.Errorf("could not return within default timeout"),
				"dep3": fmt.Errorf("ERROR"),
			},
			expectedErrMsg: []string{"deployment dep2 failed due to could not return within default timeout",
				"deployment dep3 failed due to ERROR"},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			syncMap := &sync.Map{}
			for k, v := range test.deps {
				syncMap.Store(k, v)
			}
			err := getSkaffoldDeployStatus(syncMap)
			t.CheckError(test.shouldErr, err)
			for _, msg := range test.expectedErrMsg {
				t.CheckErrorContains(msg, err)
			}
		})
	}
}

func TestGetRollOutStatus(t *testing.T) {
	rolloutCmd := "kubectl --context kubecontext --namespace test rollout status deployment dep --watch=false"
	tests := []struct {
		description string
		command     util.Command
		expected    string
		shouldErr   bool
	}{
		{
			description: "some output",
			command: testutil.NewFakeCmd(t).
				WithRunOut(rolloutCmd, "Waiting for replicas to be available"),
			expected: "Waiting for replicas to be available",
		},
		{
			description: "no output",
			command: testutil.NewFakeCmd(t).
				WithRunOut(rolloutCmd, ""),
		},
		{
			description: "rollout status error",
			command: testutil.NewFakeCmd(t).
				WithRunOutErr(rolloutCmd, "", fmt.Errorf("error")),
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			cli := &kubectl.CLI{KubeContext: testKubeContext, Namespace: "test"}
			actual, err := getRollOutStatus(context.Background(), cli, "dep")
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}
