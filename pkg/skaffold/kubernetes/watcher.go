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
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return func() {}, errors.Wrap(err, "getting k8s client")
	}

	var forever int64 = 3600 * 24 * 365 * 100

	for _, ns := range namespaces {
		watcher, err := kubeclient.CoreV1().Pods(ns).Watch(meta_v1.ListOptions{
			IncludeUninitialized: true,
			TimeoutSeconds:       &forever,
		})
		if err != nil {
			stopWatchers()
			return func() {}, errors.Wrap(err, "initializing pod watcher for "+ns)
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
