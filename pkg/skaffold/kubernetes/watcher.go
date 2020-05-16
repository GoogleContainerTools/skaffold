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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// AggregatePodWatcher returns a watcher for multiple namespaces.
func AggregatePodWatcher(namespaces []string, aggregate chan<- watch.Event) (func(), error) {
	watchers := make([]watch.Interface, 0, len(namespaces))
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

	for _, ns := range namespaces {
		watcher, err := kubeclient.CoreV1().Pods(ns).Watch(metav1.ListOptions{
			TimeoutSeconds: &forever,
		})
		if err != nil {
			stopWatchers()
			return func() {}, fmt.Errorf("initializing pod watcher for %q: %w", ns, err)
		}

		watchers = append(watchers, watcher)

		go func() {
			for msg := range watcher.ResultChan() {
				aggregate <- msg
			}
		}()
	}

	return stopWatchers, nil
}
