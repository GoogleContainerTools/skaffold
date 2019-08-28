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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resources"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	utilpointer "k8s.io/utils/pointer"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFetchDeployments(t *testing.T) {
	labeller := NewLabeller("")
	tests := []struct {
		description string
		deps        []*appsv1.Deployment
		expected    []*resources.Deployment
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
							RunIDLabel: labeller.runID,
							"random":   "foo",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(10)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(20)},
				},
			},
			expected: []*resources.Deployment{
				resources.NewDeployment("dep1", "test", 10*time.Second),
				resources.NewDeployment("dep2", "test", 20*time.Second),
			},
		}, {
			description: "command flag deadline is less than deployment spec.",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
							"random":   "foo",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(300)},
				},
			},
			expected: []*resources.Deployment{resources.NewDeployment("dep1", "test", 200*time.Second)},
		},
		{
			description: "multiple deployments with 1 no progress deadline set",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
				},
			},
			expected: []*resources.Deployment{
				resources.NewDeployment("dep1", "test", 100*time.Second),
				resources.NewDeployment("dep2", "test", 200*time.Second),
			},
		},
		{
			description: "no deployments",
			expected:    []*resources.Deployment{},
		},
		{
			description: "multiple deployments in different namespaces",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep2",
						Namespace: "test1",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
			},
			expected: []*resources.Deployment{
				resources.NewDeployment("dep1", "test", 100*time.Second),
			},
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
			expected: []*resources.Deployment{},
		},
		{
			description: "deployment in correct namespace deployed by skaffold but different run",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: "9876-6789",
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(100)},
				},
			},
			expected: []*resources.Deployment{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.deps))
			for i, dep := range test.deps {
				objs[i] = dep
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)

			actual, err := fetchDeployments(client, "test", labeller, 200)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual,
				cmp.AllowUnexported(resources.Deployment{}, resources.ResourceObj{}, resources.Status{}))
		})
	}
}

func TestIsSkaffoldDeployInError(t *testing.T) {
	var tests = []struct {
		description string
		resources   []Resource
		shouldErr   bool
	}{
		{
			description: "one error",
			resources: []Resource{
				resources.NewDeployment("dep1", "test", time.Minute),
				resources.NewDeployment("dep2", "test", time.Minute).WithError(fmt.Errorf("could not return within default timeout")),
			},
			shouldErr: true,
		},
		{
			description: "no error",
			resources: []Resource{
				resources.NewDeployment("dep1", "test", time.Minute),
			},
		},
		{
			description: "multiple errors",
			resources: []Resource{
				resources.NewDeployment("dep1", "test", time.Minute),
				resources.NewDeployment("dep2", "test", time.Minute).WithError(fmt.Errorf("could not return within default timeout")),
				resources.NewDeployment("dep3", "test", time.Minute).WithError(fmt.Errorf("err")),
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			checker := Checker{}
			t.CheckDeepEqual(test.shouldErr, checker.isSkaffoldDeployInError(test.resources))
		})
	}
}

func TestCheckResourceStatus(t *testing.T) {
	rolloutCmd := "kubectl --context kubecontext --namespace test rollout status deployment dep --watch=false"
	var tests = []struct {
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

		})
	}
}
