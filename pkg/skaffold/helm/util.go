/*
Copyright 2022 The Skaffold Authors

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

package helm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// copy of cmd/skaffold/app/flags.BuildOutputs
type buildOutputs struct {
	Builds []graph.Artifact `json:"builds"`
}

func writeBuildArtifacts(builds []graph.Artifact) (string, func(), error) {
	buildOutput, err := json.Marshal(buildOutputs{builds})
	if err != nil {
		return "", nil, fmt.Errorf("cannot marshal build artifacts: %w", err)
	}

	f, err := ioutil.TempFile("", "builds*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create temp file: %w", err)
	}
	if _, err := f.Write(buildOutput); err != nil {
		return "", nil, fmt.Errorf("cannot write to temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", nil, fmt.Errorf("cannot close temp file: %w", err)
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// SanitizeFilePath is used to sanitize filepaths that are provided to the `setFiles` flag
// helm `setFiles` doesn't work with the unescaped filepath separator (\) for Windows or if there are unescaped tabs and spaces in the directory names.
// So we escape all odd count occurrences of `\` for Windows, and wrap the entire string in quotes if it has spaces.
// This is very specific to the way helm handles its flags.
// See https://github.com/helm/helm/blob/d55c53df4e394fb62b0514a09c57bce235dd7877/pkg/cli/values/options.
// Otherwise the windows `syscall` package implements its own sanitizing for command args that's used by `exec.Cmd`.
// See https://github.com/golang/go/blob/6951da56b0ae2cd4250fc1b0350d090aed633ac1/src/syscall/exec_windows.go#L27
func SanitizeFilePath(s string, isWindowsOS bool) string {
	if len(s) == 0 {
		return `""`
	}
	needsQuotes := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			needsQuotes = true
			break
		}
	}

	if !isWindowsOS {
		if needsQuotes {
			return fmt.Sprintf(`"%s"`, s)
		}
		return s
	}

	var b []byte
	slashes := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			slashes++
		default:
			// ensure a single slash is escaped
			if slashes == 1 {
				b = append(b, '\\')
			}
			slashes = 0
		}
		b = append(b, s[i])
	}
	if slashes == 1 {
		b = append(b, '\\')
	}
	if needsQuotes {
		return fmt.Sprintf(`"%s"`, string(b))
	}
	return string(b)
}

func ChartSource(r latest.HelmRelease) string {
	if r.RemoteChart != "" {
		return r.RemoteChart
	}
	return r.ChartPath
}

func ReleaseNamespace(namespace string, release latest.HelmRelease) (string, error) {
	if namespace != "" {
		return namespace, nil
	} else if release.Namespace != "" {
		namespace, err := util.ExpandEnvTemplateOrFail(release.Namespace, nil)
		if err != nil {
			return "", fmt.Errorf("cannot parse the release namespace template: %w", err)
		}
		return namespace, nil
	}
	return "", nil
}
