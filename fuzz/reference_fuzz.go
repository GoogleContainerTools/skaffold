// +build gofuzz

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

package fuzz

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	_ "github.com/dvyukov/go-fuzz/go-fuzz-dep"
)

// FuzzParseReference tests Docker image reference parsing.
func FuzzParseReference(data []byte) int {
	if _, err := docker.ParseReference(string(data)); err != nil {
		return 0
	}
	return 1
}
