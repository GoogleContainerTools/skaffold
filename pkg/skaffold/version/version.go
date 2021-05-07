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

package version

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

var version, gitCommit, buildDate, client string
var platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

type Info struct {
	Version       string
	ConfigVersion string
	GitVersion    string
	GitCommit     string
	BuildDate     string
	GoVersion     string
	Compiler      string
	Platform      string
	User          string
}

// Get returns the version and buildtime information about the binary.
// Can be overridden for tests.
var Get = func() *Info {
	// These variables typically come from -ldflags settings to `go build`
	return &Info{
		Version:       version,
		ConfigVersion: latestV1.Version,
		GitCommit:     gitCommit,
		BuildDate:     buildDate,
		GoVersion:     runtime.Version(),
		Compiler:      runtime.Compiler,
		Platform:      platform,
	}
}

var SetClient = func(user string) {
	if _, ok := constants.AllowedUsers[user]; ok {
		client = user
	}
}

// UserAgent returns a conformant value for HTTP `User-Agent` headers.  It is of the
// form `skaffold/<version> (<os>/<arch>)`, and the version will be omitted if not available.
func UserAgent() string {
	if version == "" {
		// likely running under tests
		return fmt.Sprintf("skaffold (%s)", platform)
	}
	return fmt.Sprintf("skaffold/%s (%s)", version, platform)
}

// UserAgentWithClient returns a conformant value for HTTP `User-Agent` headers that includes
// the client value provided with the `--user` flag.  If there is no client value, then the value will be equivalent
// to `UserAgent()`.  Otherwise it is of the form `skaffold/<version> (<os>/<arch>) <client>`, and the version
// will be omitted if not available.
// Use UserAgentWithClient method to record requests from skaffold CLI users vs
// other clients.
func UserAgentWithClient() string {
	if client == "" {
		return UserAgent()
	}
	return fmt.Sprintf("%s %s", UserAgent(), client)
}

func ParseVersion(version string) (semver.Version, error) {
	// Strip the leading 'v' in our version strings
	version = strings.TrimSpace(version)
	v, err := semver.Parse(strings.TrimLeft(version, "v"))
	if err != nil {
		return semver.Version{}, fmt.Errorf("parsing semver: %w", err)
	}
	return v, nil
}
