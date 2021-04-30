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

	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
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
		ConfigVersion: latest_v1.Version,
		GitCommit:     gitCommit,
		BuildDate:     buildDate,
		GoVersion:     runtime.Version(),
		Compiler:      runtime.Compiler,
		Platform:      platform,
	}
}

var SetClient = func(user string) {
	client = user
}

func UserAgent() string {
	return fmt.Sprintf("skaffold/%s/%s", platform, version)
}

// UserAgentWithClient returns a string to be passed as the `--user` flag value
// to a request header
// Use UserAgentWithClient method to record requests from skaffold CLI users vs
// other clients.
func UserAgentWithClient() string {
	if client == "" {
		return fmt.Sprintf("skaffold/%s/%s", platform, version)
	}
	return fmt.Sprintf("skaffold-%s/%s/%s", client, platform, version)
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
