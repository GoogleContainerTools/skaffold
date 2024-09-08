/*
Copyright 2019 The Kubernetes Authors.

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

package nerdctl

import (
	"strings"

	"sigs.k8s.io/kind/pkg/exec"
)

// IsAvailable checks if nerdctl (or finch) is available in the system
func IsAvailable() bool {
	cmd := exec.Command("nerdctl", "-v")
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		// check finch
		cmd = exec.Command("finch", "-v")
		lines, err = exec.OutputLines(cmd)
		if err != nil || len(lines) != 1 {
			return false
		}
		return strings.HasPrefix(lines[0], "finch version")
	}
	return strings.HasPrefix(lines[0], "nerdctl version")
}

// rootless: use fuse-overlayfs by default
// https://github.com/kubernetes-sigs/kind/issues/2275
func mountFuse(binaryName string) bool {
	i, err := info(binaryName)
	if err != nil {
		return false
	}
	if i != nil && i.Rootless {
		return true
	}
	return false
}
