// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"os/exec"

	"github.com/spf13/cobra"
)

// AddKubeCommands augments our CLI surface with a passthru delete command, and an apply
// command that realizes the promise of ko, as outlined here:
//
//	https://github.com/google/go-containerregistry/issues/80
func AddKubeCommands(topLevel *cobra.Command) {
	addDelete(topLevel)
	addVersion(topLevel)
	addCreate(topLevel)
	addApply(topLevel)
	addResolve(topLevel)
	addBuild(topLevel)
	addRun(topLevel)
}

// check if kubectl is installed
func isKubectlAvailable() bool {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return false
	}
	return true
}
