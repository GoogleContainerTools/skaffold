/*
Copyright 2018 The Knative Authors

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

package configmap

import (
	corev1 "k8s.io/api/core/v1"
)

// Observer is the signature of the callbacks that notify an observer of the latest
// state of a particular configuration.  An observer should not modify the provided
// ConfigMap, and should `.DeepCopy()` it for persistence (or otherwise process its
// contents).
type Observer func(*corev1.ConfigMap)

// Watcher defines the interface that a configmap implementation must implement.
type Watcher interface {
	// Watch is called to register callbacks to be notified when a named ConfigMap changes.
	Watch(string, ...Observer)

	// Start is called to initiate the watches and provide a channel to signal when we should
	// stop watching.  When Start returns, all registered Observers will be called with the
	// initial state of the ConfigMaps they are watching.
	Start(<-chan struct{}) error
}

// DefaultingWatcher is similar to Watcher, but if a ConfigMap is absent, then a code provided
// default will be used.
type DefaultingWatcher interface {
	Watcher

	// WatchWithDefault is called to register callbacks to be notified when a named ConfigMap
	// changes. The provided default value is always observed before any real ConfigMap with that
	// name is. If the real ConfigMap with that name is deleted, then the default value is observed.
	WatchWithDefault(cm corev1.ConfigMap, o ...Observer)
}
