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

// Package version provides utilities for version number comparisons
//
// This is forked from k8s.io/apimachinery/pkg/util/version to make
// kind easier to import (k8s.io/apimachinery/pkg/util/version is a stable,
// mature package with no externaldependencies within a large, heavy module)
package version
