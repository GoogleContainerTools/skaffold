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

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

type PodWatcher interface {
	Register(receiver chan<- PodEvent)
	Deregister(receiver chan<- PodEvent)
	Start(kubeContext string, ns []string) (func(), error)
}

// podWatcher is a pod watcher for multiple namespaces.
type podWatcher struct {
	podSelector  PodSelector
	receivers    map[chan<- PodEvent]bool
	receiverLock sync.Mutex
}

type PodEvent struct {
	Type watch.EventType
	Pod  *v1.Pod
}

func NewPodWatcher(podSelector PodSelector) PodWatcher {
	return &podWatcher{
		podSelector: podSelector,
		receivers:   make(map[chan<- PodEvent]bool),
	}
}

func (w *podWatcher) Register(receiver chan<- PodEvent) {
	w.receiverLock.Lock()
	w.receivers[receiver] = true
	w.receiverLock.Unlock()
}

func (w *podWatcher) Deregister(receiver chan<- PodEvent) {
	w.receiverLock.Lock()
	w.receivers[receiver] = false
	w.receiverLock.Unlock()
}

func (w *podWatcher) Start(kubeContext string, namespaces []string) (func(), error) {
	if len(w.receivers) == 0 {
		return func() {}, errors.New("no receiver was registered")
	}

	var watchers []watch.Interface
	stopWatchers := func() {
		for _, w := range watchers {
			w.Stop()
		}
	}

	kubeclient, err := client.Client(kubeContext)
	if err != nil {
		return func() {}, fmt.Errorf("getting k8s client: %w", err)
	}

	var forever int64 = 3600 * 24 * 365 * 100

	for _, ns := range namespaces {
		watcher, err := kubeclient.CoreV1().Pods(ns).Watch(context.Background(), metav1.ListOptions{
			TimeoutSeconds: &forever,
		})
		if err != nil {
			stopWatchers()
			return func() {}, fmt.Errorf("initializing pod watcher for %q: %w", ns, err)
		}

		watchers = append(watchers, watcher)
		go func() {
			l := log.Entry(context.TODO())
			for evt := range watcher.ResultChan() {
				// If the event's type is "ERROR", log and continue.
				if evt.Type == watch.Error {
					// These errors sem to arise from the watch stream being closed from a ^C.
					// evt.Object seems likely to be a https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Status
					//    Status{
					//        Status:Failure,
					//        Code:500,
					//        Reason:InternalError,
					//        Message:an error on the server ("unable to decode an event from the watch stream: http2: response body closed") has prevented the request from succeeding,
					//        Details:&StatusDetails{
					//          Causes:[]StatusCause{
					//            {Type:UnexpectedServerResponse,Message:unable to decode an event from the watch stream: http2: response body closed},
					//            {Type:ClientWatchDecoding,Message:unable to decode an event from the watch stream: http2: response body closed}},
					//          RetryAfterSeconds:0}}
					l.Debugf("podWatcher: got unexpected event of type %s: %v", evt.Type, evt.Object)
					continue
				}

				// Grab the pod from the event.
				pod, ok := evt.Object.(*v1.Pod)
				if !ok {
					continue
				}

				if !w.podSelector.Select(pod) {
					continue
				}

				if l.Logger.IsLevelEnabled(logrus.TraceLevel) {
					st := fmt.Sprintf("podWatcher[%s/%s:%v] phase:%v ", pod.Namespace, pod.Name, evt.Type, pod.Status.Phase)
					if len(pod.Status.Reason) > 0 {
						st += fmt.Sprintf("reason:%s ", pod.Status.Reason)
					}
					for _, c := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
						switch {
						case c.State.Waiting != nil:
							st += fmt.Sprintf("%s<waiting> ", c.Name)
						case c.State.Running != nil:
							st += fmt.Sprintf("%s<running> ", c.Name)
						case c.State.Terminated != nil:
							st += fmt.Sprintf("%s<terminated> ", c.Name)
						}
					}
					l.Trace(st)
				}

				w.receiverLock.Lock()
				for receiver, open := range w.receivers {
					if open {
						receiver <- PodEvent{
							Type: evt.Type,
							Pod:  pod,
						}
					}
				}
				w.receiverLock.Unlock()
			}
		}()
	}

	return stopWatchers, nil
}
