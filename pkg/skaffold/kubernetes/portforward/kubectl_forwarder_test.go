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
	"bytes"
	"context"
	"errors"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUnavailablePort(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&waitPortNotFree, 100*time.Millisecond)

		// Return that the port is false, while also
		// adding a sync group so we know when isPortFree
		// has been called
		var portFreeWG sync.WaitGroup
		portFreeWG.Add(1)
		t.Override(&isPortFree, func(string, int) bool {
			portFreeWG.Done()
			return false
		})

		// Create a wait group that will only be
		// fulfilled when the forward function returns
		var forwardFunctionWG sync.WaitGroup
		forwardFunctionWG.Add(1)
		t.Override(&deferFunc, func() {
			forwardFunctionWG.Done()
		})

		var buf bytes.Buffer
		k := KubectlForwarder{
			out: &buf,
		}
		pfe := newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false)

		go k.Forward(context.Background(), pfe)

		// wait for isPortFree to be called
		portFreeWG.Wait()

		// then, end port forwarding and wait for the forward function to return.
		pfe.terminationLock.Lock()
		pfe.terminated = true
		pfe.terminationLock.Unlock()
		forwardFunctionWG.Wait()

		// read output to make sure logs are expected
		t.CheckContains("port 8080 is taken", buf.String())
	})
}

func TestTerminate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pfe := newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false)
	pfe.cancel = cancel

	k := &KubectlForwarder{}
	k.Terminate(pfe)
	if pfe.terminated != true {
		t.Fatalf("expected pfe.terminated to be true after termination")
	}
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected cancel to be called")
	}
}

func TestMonitorErrorLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip flaky test until it's fixed")
	}
	tests := []struct {
		description string
		input       string
		cmdRunning  bool
		shouldError bool
	}{
		{
			description: "no error logs appear",
			input:       "some random logs",
			cmdRunning:  true,
		},
		{
			description: "match on 'error forwarding port'",
			input:       "error forwarding port 8080",
			shouldError: true,
		},
		{
			description: "match on 'unable to forward'",
			input:       "unable to forward 8080",
			shouldError: true,
		},
		{
			description: "match on 'error upgrading connection'",
			input:       "error upgrading connection 8080",
			shouldError: true,
		},
		{
			description: "match on successful port forwarding message",
			input:       "Forwarding from 127.0.0.1:8080 -> 8080",
			cmdRunning:  true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&waitErrorLogs, 10*time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			cmdStr := "sleep"
			if runtime.GOOS == "windows" {
				cmdStr = "timeout"
			}
			cmd := kubectl.CommandContext(ctx, cmdStr, "5")
			if err := cmd.Start(); err != nil {
				t.Fatalf("error starting command: %v", err)
			}

			errChan := make(chan error, 1)
			go func() {
				logs := strings.NewReader(test.input)

				k := KubectlForwarder{}
				k.monitorLogs(ctx, logs, cmd, &portForwardEntry{}, errChan)

				errChan <- nil
			}()

			err := <-errChan
			t.CheckError(test.shouldError, err)

			// make sure the command is running or killed based on what's expected
			if test.cmdRunning {
				assertCmdIsRunning(t, cmd)
				cmd.Terminate()
			} else {
				assertCmdWasKilled(t, cmd)
			}
		})
	}
}

func assertCmdIsRunning(t *testutil.T, cmd *kubectl.Cmd) {
	if cmd.ProcessState != nil {
		t.Fatal("cmd was killed but expected to continue running")
	}
}

func assertCmdWasKilled(t *testutil.T, cmd *kubectl.Cmd) {
	if err := cmd.Wait(); err == nil {
		t.Fatal("cmd was not killed but expected to be killed")
	}
}

func TestPortForwardArgs(t *testing.T) {
	tests := []struct {
		description string
		input       *portForwardEntry
		servicePod  string
		servicePort int
		serviceErr  error
		result      []string
	}{
		{
			description: "non-default address",
			input:       newPortForwardEntry(0, latest.PortForwardResource{Type: "pod", Name: "p", Namespace: "ns", Port: schemautil.FromInt(9), Address: "0.0.0.0"}, "", "", "", "", 8080, false),
			result:      []string{"--pod-running-timeout", "1s", "--namespace", "ns", "pod/p", "8080:9", "--address", "0.0.0.0"},
		},
		{
			description: "localhost is the default",
			input:       newPortForwardEntry(0, latest.PortForwardResource{Type: "pod", Name: "p", Namespace: "ns", Port: schemautil.FromInt(9), Address: "127.0.0.1"}, "", "", "", "", 8080, false),
			result:      []string{"--pod-running-timeout", "1s", "--namespace", "ns", "pod/p", "8080:9"},
		},
		{
			description: "no address",
			input:       newPortForwardEntry(0, latest.PortForwardResource{Type: "pod", Name: "p", Namespace: "ns", Port: schemautil.FromInt(9)}, "", "", "", "", 8080, false),
			result:      []string{"--pod-running-timeout", "1s", "--namespace", "ns", "pod/p", "8080:9"},
		},
		{
			description: "service to pod",
			input:       newPortForwardEntry(0, latest.PortForwardResource{Type: "service", Name: "svc", Namespace: "ns", Port: schemautil.FromInt(9)}, "", "", "", "", 8080, false),
			servicePod:  "servicePod",
			servicePort: 9999,
			result:      []string{"--pod-running-timeout", "1s", "--namespace", "ns", "pod/servicePod", "8080:9999"},
		},
		{
			description: "service could not be mapped to pod",
			input:       newPortForwardEntry(0, latest.PortForwardResource{Type: "service", Name: "svc", Namespace: "ns", Port: schemautil.FromInt(9)}, "", "", "", "", 8080, false),
			serviceErr:  errors.New("error"),
			result:      []string{"--pod-running-timeout", "1s", "--namespace", "ns", "service/svc", "8080:9"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			t.Override(&findNewestPodForSvc, func(ctx context.Context, ns, serviceName string, servicePort schemautil.IntOrString) (string, int, error) {
				return test.servicePod, test.servicePort, test.serviceErr
			})

			args := portForwardArgs(ctx, test.input)
			t.CheckDeepEqual(test.result, args)
		})
	}
}

func TestNewestPodFirst(t *testing.T) {
	starting := mockPod("starting", nil, time.Now())
	starting.Status.Phase = corev1.PodPending
	new := mockPod("new", nil, time.Now().Add(-time.Minute))
	old := mockPod("old", nil, time.Now().Add(-time.Hour))

	pods := []corev1.Pod{*old, *new, *starting}
	sort.Slice(pods, newestPodsFirst(pods))

	expected := []corev1.Pod{*starting, *new, *old}
	testutil.CheckDeepEqual(t, expected, pods)
}

func TestFindServicePort(t *testing.T) {
	tests := []struct {
		description string
		service     *corev1.Service
		port        schemautil.IntOrString
		shouldErr   bool
		expected    corev1.ServicePort
	}{
		{
			description: "simple case",
			service:     mockService("svc1", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 90, TargetPort: intstr.FromInt(80)}, {Port: 80, TargetPort: intstr.FromInt(8080)}}),
			port:        schemautil.FromInt(80),
			expected:    corev1.ServicePort{Port: 80, TargetPort: intstr.FromInt(8080)},
		},
		{
			description: "no ports",
			service:     mockService("svc2", corev1.ServiceTypeLoadBalancer, nil),
			port:        schemautil.FromInt(80),
			shouldErr:   true,
		},
		{
			description: "no matching ports",
			service:     mockService("svc3", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 90, TargetPort: intstr.FromInt(80)}, {Port: 80, TargetPort: intstr.FromInt(8080)}}),
			port:        schemautil.FromInt(100),
			shouldErr:   true,
		},
		{
			description: "simple case with service port names",
			service:     mockService("svc1", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Name: "aaa", Port: 90, TargetPort: intstr.FromInt(80)}, {Name: "bbb", Port: 80, TargetPort: intstr.FromInt(8080)}}),
			port:        schemautil.FromString("bbb"),
			expected:    corev1.ServicePort{Name: "bbb", Port: 80, TargetPort: intstr.FromInt(8080)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result, err := findServicePort(*test.service, test.port)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, result)
		})
	}
}

func TestFindTargetPort(t *testing.T) {
	tests := []struct {
		description string
		servicePort corev1.ServicePort
		pod         corev1.Pod
		expected    int
	}{
		{
			description: "integer port",
			servicePort: corev1.ServicePort{TargetPort: intstr.FromInt(8080)},
			pod:         *mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Time{}),
			expected:    8080,
		},
		{
			description: "named port",
			servicePort: corev1.ServicePort{TargetPort: intstr.FromString("http")},
			pod:         *mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Time{}),
			expected:    8080,
		},
		{
			description: "no port found",
			servicePort: corev1.ServicePort{TargetPort: intstr.FromString("http")},
			pod:         *mockPod("new", nil, time.Time{}),
			expected:    -1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			result := findTargetPort(test.servicePort, test.pod)
			t.CheckDeepEqual(test.expected, result)
		})
	}
}

func TestFindNewestPodForService(t *testing.T) {
	tests := []struct {
		description     string
		clientResources []pkgruntime.Object
		clientErr       error
		serviceName     string
		servicePort     int
		shouldErr       bool
		chosenPod       string
		chosenPort      int
	}{
		{
			description: "chooses new with port 8080 via int targetport",
			clientResources: []pkgruntime.Object{
				mockService("svc", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}}),
				mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Minute)),
				mockPod("old", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Hour)),
			},
			serviceName: "svc",
			servicePort: 80,
			chosenPod:   "new",
			chosenPort:  8080,
		},
		{
			description: "chooses new with port 8080 via string targetport",
			clientResources: []pkgruntime.Object{
				mockService("svc", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromString("http")}}),
				mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Minute)),
				mockPod("old", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Hour)),
			},
			serviceName: "svc",
			servicePort: 80,
			chosenPod:   "new",
			chosenPort:  8080,
		},
		{
			description: "service not found",
			clientResources: []pkgruntime.Object{
				mockService("svc", corev1.ServiceTypeClusterIP, []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}}),
				mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Minute)),
				mockPod("old", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Hour)),
			},
			serviceName: "notfound",
			servicePort: 80,
			shouldErr:   true,
			chosenPort:  -1,
		},
		{
			description: "port not found",
			clientResources: []pkgruntime.Object{
				mockService("svc", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}}),
				mockPod("new", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Minute)),
				mockPod("old", []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}, time.Now().Add(-time.Hour)),
			},
			serviceName: "svc",
			servicePort: 90,
			shouldErr:   true,
			chosenPort:  -1,
		},
		{
			description: "no matching pods",
			clientResources: []pkgruntime.Object{
				mockService("service", corev1.ServiceTypeLoadBalancer, []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}}),
			},
			serviceName: "svc",
			servicePort: 90,
			shouldErr:   true,
			chosenPort:  -1,
		},
		{
			description: "port not found",
			clientErr:   errors.New("injected failure"),
			serviceName: "svc",
			servicePort: 90,
			shouldErr:   true,
			chosenPort:  -1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			t.Override(&client.Client, func() (kubernetes.Interface, error) {
				return fake.NewSimpleClientset(test.clientResources...), test.clientErr
			})

			pod, port, err := findNewestPodForService(ctx, "", test.serviceName, schemautil.FromInt(test.servicePort))
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.chosenPod, pod)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.chosenPort, port)
		})
	}
}

func mockService(name string, serviceType corev1.ServiceType, ports []corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.ServiceSpec{
			Type:  serviceType,
			Ports: ports,
		}}
}

func mockPod(name string, ports []corev1.ContainerPort, creationTime time.Time) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "container",
				Ports: ports,
			}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
}
