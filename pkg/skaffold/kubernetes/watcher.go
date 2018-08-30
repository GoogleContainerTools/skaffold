/*
Copyright 2018 The Skaffold Authors

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

// PodWatcher returns a watcher that will report on all Pod Events (additions, modifications, etc.)
func PodWatcher() (watch.Interface, error) {
	kubeclient, err := Client()
	if err != nil {
		return nil, errors.Wrap(err, "getting k8s client")
	}
	client := kubeclient.CoreV1()
	var forever int64 = 3600 * 24 * 365 * 100
	return client.Pods("").Watch(meta_v1.ListOptions{
		IncludeUninitialized: true,
		TimeoutSeconds:       &forever,
	})
}
