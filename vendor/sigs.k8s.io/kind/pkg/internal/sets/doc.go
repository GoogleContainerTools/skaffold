/*
Copyright 2021 The Kubernetes Authors.

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

// Package sets implements set types.
//
// This is forked from k8s.io/apimachinery/pkg/util/sets (under the same project
// and license), because k8s.io/apimachinery is a relatively heavy dependency
// and we only need some trivial utilities. Avoiding importing k8s.io/apimachinery
// makes kind easier to embed in other projects for testing etc.
//
// The set implementation is relatively small and very stable.
package sets
