/*
Copyright 2020 The Skaffold Authors

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

package buildpacks

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"

var debugModeArgs = map[string]string{
	"GOOGLE_GOGCFLAGS": "all=-N -l", // disable build optimization for Golang
	// TODO: Add for other languages
}

var nonDebugModeArgs = map[string]string{}

func addDefaultArgs(mode config.RunMode, existing map[string]string) map[string]string {
	var args map[string]string
	switch mode {
	case config.RunModes.Debug:
		args = debugModeArgs
	default:
		args = nonDebugModeArgs
	}

	for k, v := range args {
		if _, found := existing[k]; !found {
			existing[k] = v
		}
	}
	return existing
}
