/*
Copyright 2018 Knative Authors LLC
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

package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	buildv1alpha1 "github.com/knative/build/pkg/client/clientset/versioned/typed/build/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const buildExecuteFailed = "BuildExecuteFailed"

// Tail tails the logs for a build.
func Tail(ctx context.Context, out io.Writer, buildName, namespace string) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	cfg, err := kubeConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("getting clientConfig: %v", err)
	}

	podName, err := podName(cfg, out, buildName, namespace)
	if err != nil {
		return fmt.Errorf("getting build pod: %v", err)
	}

	client, err := corev1.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("getting kubernetes client: %v", err)
	}

	pods := client.Pods(namespace)
	watcher := podWatcher{
		pods: pods,
		name: podName,
	}
	if err := watcher.start(ctx); err != nil {
		return fmt.Errorf("watching pod: %v", err)
	}

	pod, err := watcher.waitForPod(ctx, func(p *v1.Pod) bool {
		return len(p.Status.InitContainerStatuses) > 0
	})
	if err != nil {
		return err
	}

	for i, container := range pod.Status.InitContainerStatuses {
		pod, err := watcher.waitForPod(ctx, func(p *v1.Pod) bool {
			waiting := p.Status.InitContainerStatuses[i].State.Waiting
			if waiting == nil {
				return true
			}

			if waiting.Message != "" {
				fmt.Fprintln(out, red(fmt.Sprintf("[%s] %s", container.Name, waiting.Message)))
			}

			return false
		})
		if err != nil {
			return fmt.Errorf("waiting for container: %v", err)
		}

		container := pod.Status.InitContainerStatuses[i]
		followContainer := container.State.Terminated == nil
		if err := printContainerLogs(ctx, out, pods, podName, container.Name, followContainer); err != nil {
			return fmt.Errorf("printing logs: %v", err)
		}

		pod, err = watcher.waitForPod(ctx, func(p *v1.Pod) bool {
			return p.Status.InitContainerStatuses[i].State.Terminated != nil
		})
		if err != nil {
			return fmt.Errorf("waiting for container termination: %v", err)
		}

		container = pod.Status.InitContainerStatuses[i]
		terminated := container.State.Terminated
		if terminated.ExitCode != 0 {
			message := "Build Failed"
			if terminated.Message != "" {
				message += ": " + terminated.Message
			}

			fmt.Fprintln(out, red(fmt.Sprintf("[%s] %s", container.Name, message)))
			return nil
		}
	}

	return nil
}

type podWatcher struct {
	pods corev1.PodInterface
	name string

	versions chan *v1.Pod
	last     *v1.Pod
}

func (w *podWatcher) start(ctx context.Context) error {
	w.versions = make(chan *v1.Pod, 100)

	watcher, err := w.pods.Watch(metav1.ListOptions{
		IncludeUninitialized: true,
		FieldSelector:        fields.OneTermEqualSelector("metadata.name", w.name).String(),
	})
	if err != nil {
		return fmt.Errorf("watching pod: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				watcher.Stop()
				return
			case evt, ok := <-watcher.ResultChan():
				if !ok {
					// TODO: reconnect watch
				}

				w.versions <- evt.Object.(*v1.Pod)
			}
		}
	}()

	return nil
}

func (w *podWatcher) waitForPod(ctx context.Context, predicate func(pod *v1.Pod) bool) (*v1.Pod, error) {
	if w.last != nil && predicate(w.last) {
		return w.last, nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("watch cancelled")
		case w.last = <-w.versions:
			if predicate(w.last) {
				return w.last, nil
			}
		}
	}
}

func printContainerLogs(ctx context.Context, out io.Writer, pods corev1.PodExpansion, podName, containerName string, follow bool) error {
	rc, err := pods.GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
	}).Stream()
	if err != nil {
		return err
	}
	defer rc.Close()

	return streamLogs(ctx, out, containerName, rc)
}

func streamLogs(ctx context.Context, out io.Writer, containerName string, rc io.Reader) error {
	prefix := green(fmt.Sprintf("[%s]", containerName)) + " "

	r := bufio.NewReader(rc)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			if len(line) > 0 {
				fmt.Fprintf(out, "%s%s\n", prefix, line)
			}
			return nil
		}
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "%s%s", prefix, line)
	}
}

// podName finds the pod name for a given build.
func podName(cfg *rest.Config, out io.Writer, buildName, namespace string) (string, error) {
	client, err := buildv1alpha1.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("getting build client: %v", err)
	}

	for ; ; time.Sleep(time.Second) {
		b, err := client.Builds(namespace).Get(buildName, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			return "", fmt.Errorf("getting build: %v", err)
		}

		cluster := b.Status.Cluster
		if cluster != nil && cluster.PodName != "" {
			return cluster.PodName, nil
		}

		for _, condition := range b.Status.Conditions {
			if condition.Reason == buildExecuteFailed {
				return "", fmt.Errorf("build failed: %s", condition.Message)
			}
		}
	}
}

func green(text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 32, text)
}

func red(text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", 31, text)
}
