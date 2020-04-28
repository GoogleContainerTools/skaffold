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
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	utilpointer "k8s.io/utils/pointer"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/resource"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetDeployments(t *testing.T) {
	labeller := NewLabeller(config.SkaffoldOptions{})
	tests := []struct {
		description string
		deps        []*appsv1.Deployment
		expected    []Resource
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
			expected: []Resource{
				resource.NewDeployment("dep1", "test", 10*time.Second),
				resource.NewDeployment("dep2", "test", 20*time.Second),
			},
		},
		{
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
			expected: []Resource{
				resource.NewDeployment("dep1", "test", 300*time.Second),
			},
		},
		{
			description: "multiple deployments with no progress deadline set",
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
			expected: []Resource{
				resource.NewDeployment("dep1", "test", 100*time.Second),
				resource.NewDeployment("dep2", "test", 200*time.Second),
			},
		},
		{
			description: "multiple deployments with progress deadline set to max",
			deps: []*appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep1",
						Namespace: "test",
						Labels: map[string]string{
							RunIDLabel: labeller.runID,
						},
					},
					Spec: appsv1.DeploymentSpec{ProgressDeadlineSeconds: utilpointer.Int32Ptr(600)},
				},
			},
			expected: []Resource{
				resource.NewDeployment("dep1", "test", 200*time.Second),
			},
		},
		{
			description: "no deployments",
			expected:    []Resource{},
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
			expected: []Resource{
				resource.NewDeployment("dep1", "test", 100*time.Second),
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
			expected: []Resource{},
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
			expected: []Resource{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.deps))
			for i, dep := range test.deps {
				objs[i] = dep
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)
			actual, err := getDeployments(client, "test", labeller, 200*time.Second)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, &test.expected, &actual,
				cmp.AllowUnexported(resource.Base{}, resource.Deployment{}, resource.Status{}))
		})
	}
}

type mockResource struct {
	*resource.Base
	inErr bool
	done  bool
}

func (m *mockResource) UpdateStatus(s string, err error) {
	err = errors.Unwrap(err)
	if err == context.DeadlineExceeded {
		m.inErr = true
	}
}

func (m *mockResource) Deadline() time.Duration {
	return 5 * time.Millisecond
}

func (m *mockResource) CheckStatus(context.Context, *runcontext.RunContext) {
}

func (m *mockResource) IsStatusCheckComplete() bool {
	return m.done
}

func TestPollResourceStatus(t *testing.T) {
	tests := []struct {
		description   string
		dummyResource *mockResource
		isInErr       bool
	}{
		{
			description:   "resource never stabilize within deadline",
			dummyResource: &mockResource{},
			isInErr:       true,
		},
		{
			description:   "resource stabilizes",
			dummyResource: &mockResource{done: true},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&defaultPollPeriodInMilliseconds, 0)
			pollResourceStatus(context.Background(), nil, test.dummyResource)
			t.CheckDeepEqual(test.isInErr, test.dummyResource.inErr)
		})
	}
}

func TestGetDeployStatus(t *testing.T) {
	tests := []struct {
		description string
		counter     *counter
		expected    string
		shouldErr   bool
	}{
		{
			description: "one error",
			counter:     &counter{total: 2, failed: 1},
			expected:    "1/2 deployment(s) failed",
			shouldErr:   true,
		},
		{
			description: "no error",
			counter:     &counter{total: 2},
		},
		{
			description: "multiple errors",
			counter:     &counter{total: 3, failed: 2},
			expected:    "2/3 deployment(s) failed",
			shouldErr:   true,
		},
		{
			description: "0 deployments",
			counter:     &counter{},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			event.InitializeState(latest.Pipeline{}, "test")
			err := getSkaffoldDeployStatus(test.counter)
			t.CheckError(test.shouldErr, err)
			if test.shouldErr {
				t.CheckErrorContains(test.expected, err)
			}
		})
	}
}

func TestPrintSummaryStatus(t *testing.T) {
	tests := []struct {
		description string
		namespace   string
		deployment  string
		pending     int32
		err         error
		expected    string
	}{
		{
			description: "no deployment left and current is in success",
			namespace:   "test",
			deployment:  "dep",
			pending:     0,
			err:         nil,
			expected:    " - test:deployment/dep is ready.\n",
		},
		{
			description: "default namespace",
			namespace:   "default",
			deployment:  "dep",
			pending:     0,
			err:         nil,
			expected:    " - deployment/dep is ready.\n",
		},
		{
			description: "no deployment left and current is in error",
			namespace:   "test",
			deployment:  "dep",
			pending:     0,
			err:         errors.New("context deadline expired"),
			expected:    " - test:deployment/dep failed. Error: context deadline expired.\n",
		},
		{
			description: "more than 1 deployment left and current is in success",
			namespace:   "test",
			deployment:  "dep",
			pending:     4,
			err:         nil,
			expected:    " - test:deployment/dep is ready. [4/10 deployment(s) still pending]\n",
		},
		{
			description: "more than 1 deployment left and current is in error",
			namespace:   "test",
			deployment:  "dep",
			pending:     8,
			err:         errors.New("context deadline expired"),
			expected:    " - test:deployment/dep failed. [8/10 deployment(s) still pending] Error: context deadline expired.\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := new(bytes.Buffer)
			rc := newResourceCounter(10)
			rc.deployments.pending = test.pending
			printStatusCheckSummary(
				out,
				withStatus(resource.NewDeployment(test.deployment, test.namespace, 0), "", test.err),
				*rc,
			)
			t.CheckDeepEqual(test.expected, out.String())
		})
	}
}

func TestPrintStatus(t *testing.T) {
	tests := []struct {
		description string
		rs          []Resource
		expectedOut string
		expected    bool
	}{
		{
			description: "single resource successful marked complete - skip print",
			rs: []Resource{
				withStatus(
					resource.NewDeployment("r1", "test", 1),
					"deployment successfully rolled out",
					nil,
				),
			},
			expected: true,
		},
		{
			description: "single resource in error marked complete -skip print",
			rs: []Resource{
				withStatus(
					resource.NewDeployment("r1", "test", 1),
					"error",
					errors.New("error"),
				),
			},
			expected: true,
		},
		{
			description: "multiple resources 1 not complete",
			rs: []Resource{
				withStatus(
					resource.NewDeployment("r1", "test", 1),
					"deployment successfully rolled out",
					nil,
				),
				withStatus(
					resource.NewDeployment("r2", "test", 1),
					"pending",
					nil,
				),
			},
			expectedOut: " - test:deployment/r2: pending\n",
		},
		{
			description: "multiple resources 1 not complete and retry-able error",
			rs: []Resource{
				withStatus(
					resource.NewDeployment("r1", "test", 1),
					"eployment successfully rolled out",
					nil,
				),
				withStatus(
					resource.NewDeployment("r2", "test", 1),
					"",
					resource.ErrKubectlConnection,
				),
			},
			expectedOut: " - test:deployment/r2: kubectl connection error\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			out := new(bytes.Buffer)
			actual := printStatus(test.rs, out)
			t.CheckDeepEqual(test.expectedOut, out.String())
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func withStatus(d *resource.Deployment, details string, err error) *resource.Deployment {
	d.UpdateStatus(details, err)
	return d
}

func TestCounterCopy(t *testing.T) {
	tests := []struct {
		description string
		c           *counter
		expected    counter
	}{
		{
			description: "initial counter is copied correctly ",
			c:           newCounter(10),
			expected:    *newCounter(10),
		},
		{
			description: "counter with updated pending is copied correctly",
			c:           &counter{total: 10, pending: 2},
			expected:    counter{total: 10, pending: 2},
		},
		{
			description: "counter with updated failed and pending is copied correctly",
			c:           &counter{total: 10, pending: 5, failed: 3},
			expected:    counter{total: 10, pending: 5, failed: 3},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.c.copy(), cmp.AllowUnexported(counter{}))
		})
	}
}

func TestResourceMarkProcessed(t *testing.T) {
	tests := []struct {
		description string
		c           *resourceCounter
		err         error
		expected    resourceCounter
	}{
		{
			description: "when deployment failed, counter is updated",
			c:           newResourceCounter(10),
			err:         errors.New("some err"),
			expected: resourceCounter{
				deployments: &counter{total: 10, failed: 1, pending: 9},
				pods:        newCounter(0),
			},
		},
		{
			description: "when deployment is successful, counter is updated",
			c:           newResourceCounter(10),
			expected: resourceCounter{
				deployments: &counter{total: 10, failed: 0, pending: 9},
				pods:        newCounter(0),
			},
		},
		{
			description: "counter when 1 deployment is updated correctly",
			c:           newResourceCounter(1),
			expected: resourceCounter{
				deployments: &counter{total: 1, failed: 0, pending: 0},
				pods:        newCounter(0),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.c.markProcessed(test.err), cmp.AllowUnexported(resourceCounter{}, counter{}))
		})
	}
}

func TestGetStatusCheckDeadline(t *testing.T) {
	tests := []struct {
		description string
		value       int
		deps        []Resource
		expected    time.Duration
	}{
		{
			description: "no value specified",
			deps: []Resource{
				resource.NewDeployment("dep1", "test", 10*time.Second),
				resource.NewDeployment("dep2", "test", 20*time.Second),
			},
			expected: 20 * time.Second,
		},
		{
			description: "value specified less than all other resources",
			value:       5,
			deps: []Resource{
				resource.NewDeployment("dep1", "test", 10*time.Second),
				resource.NewDeployment("dep2", "test", 20*time.Second),
			},
			expected: 5 * time.Second,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, statusCheckMaxDeadline(test.value, test.deps))
		})
	}
}
