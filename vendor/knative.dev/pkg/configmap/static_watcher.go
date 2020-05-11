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
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// NewStaticWatcher returns an StaticWatcher that exposes a collection of ConfigMaps.
func NewStaticWatcher(cms ...*corev1.ConfigMap) *StaticWatcher {
	cmm := make(map[string]*corev1.ConfigMap)
	for _, cm := range cms {
		cmm[cm.Name] = cm
	}
	return &StaticWatcher{cfgs: cmm}
}

// StaticWatcher is a Watcher with static ConfigMaps. Callbacks will
// occur when Watch is invoked for a specific Observer
type StaticWatcher struct {
	cfgs map[string]*corev1.ConfigMap
}

// Asserts that fixedImpl implements Watcher.
var _ Watcher = (*StaticWatcher)(nil)

// Watch implements Watcher
func (di *StaticWatcher) Watch(name string, o ...Observer) {
	cm, ok := di.cfgs[name]
	if ok {
		for _, observer := range o {
			observer(cm)
		}
	} else {
		panic(fmt.Sprintf("Tried to watch unknown config with name %q", name))
	}
}

// Start implements Watcher
func (di *StaticWatcher) Start(<-chan struct{}) error {
	return nil
}
