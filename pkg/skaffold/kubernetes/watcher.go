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
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type PodWatcher interface {
	Register(receiver chan<- PodEvent)
	Start() (func(), error)
}

// podWatcher is a pod watcher for multiple namespaces.
type podWatcher struct {
	podSelector PodSelector
	namespaces  []string
	receivers   []chan<- PodEvent
}

type PodEvent struct {
	Type watch.EventType
	Pod  *v1.Pod
}

func NewPodWatcher(podSelector PodSelector, namespaces []string) PodWatcher {
	return &podWatcher{
		podSelector: podSelector,
		namespaces:  namespaces,
	}
}

func (w *podWatcher) Register(receiver chan<- PodEvent) {
	w.receivers = append(w.receivers, receiver)
}

func (w *podWatcher) Start() (func(), error) {
	if len(w.receivers) == 0 {
		return func() {}, errors.New("no receiver was registered")
	}

	var watchers []watch.Interface
	stopWatchers := func() {
		for _, w := range watchers {
			w.Stop()
		}
	}

	kubeclient, err := Client()
	if err != nil {
		return func() {}, fmt.Errorf("getting k8s client: %w", err)
	}

	var forever int64 = 3600 * 24 * 365 * 100

	for _, ns := range w.namespaces {
		watcher, err := kubeclient.CoreV1().Pods(ns).Watch(metav1.ListOptions{
			TimeoutSeconds: &forever,
		})
		if err != nil {
			stopWatchers()
			return func() {}, fmt.Errorf("initializing pod watcher for %q: %w", ns, err)
		}

		watchers = append(watchers, watcher)

		go func() {
			for evt := range watcher.ResultChan() {
				// If the event's type is "ERROR", warn and continue.
				if evt.Type == watch.Error {
					logrus.Warnf("got unexpected event of type %s", evt.Type)
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

				for _, receiver := range w.receivers {
					receiver <- PodEvent{
						Type: evt.Type,
						Pod:  pod,
					}
				}
			}
		}()
	}

	return stopWatchers, nil
}
