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
package version

import (
	"strings"

	"github.com/blang/semver"
)

// VersionPrefix is the prefix of the git tag for a version
const VersionPrefix = "v"

// The current version of the minikube

// version is a private field and should be set when compiling with --ldflags="-X github.com/GoogleContainerTools/skaffold/pkg/diag/version.version=vX.Y.Z"
var version = "v0.0.0-unset"

// GetVersion returns the current diag pkg version
func GetVersion() string {
	return version
}

// GetSemverVersion returns the current semantic version (semver)
func GetSemverVersion() (semver.Version, error) {
	return semver.Make(strings.TrimPrefix(GetVersion(), VersionPrefix))
}
