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

package kubeconfig

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/kind/pkg/internal/sets"
)

const kubeconfigEnv = "KUBECONFIG"

/*
paths returns the list of paths to be considered for kubeconfig files
where explicitPath is the value of --kubeconfig

# Logic based on kubectl

https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands

- If the --kubeconfig flag is set, then only that file is loaded. The flag may only be set once and no merging takes place.

- If $KUBECONFIG environment variable is set, then it is used as a list of paths (normal path delimiting rules for your system). These paths are merged. When a value is modified, it is modified in the file that defines the stanza. When a value is created, it is created in the first file that exists. - If no files in the chain exist, then it creates the last file in the list.

- Otherwise, ${HOME}/.kube/config is used and no merging takes place.
*/
func paths(explicitPath string, getEnv func(string) string) []string {
	if explicitPath != "" {
		return []string{explicitPath}
	}

	paths := discardEmptyAndDuplicates(
		filepath.SplitList(getEnv(kubeconfigEnv)),
	)
	if len(paths) != 0 {
		return paths
	}

	return []string{path.Join(homeDir(runtime.GOOS, getEnv), ".kube", "config")}
}

// pathForMerge returns the file that kubectl would merge into
func pathForMerge(explicitPath string, getEnv func(string) string) string {
	// find the first file that exists
	p := paths(explicitPath, getEnv)
	if len(p) == 1 {
		return p[0]
	}
	for _, filename := range p {
		if fileExists(filename) {
			return filename
		}
	}
	// otherwise the last file
	return p[len(p)-1]
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func discardEmptyAndDuplicates(paths []string) []string {
	seen := sets.NewString()
	kept := 0
	for _, p := range paths {
		if p != "" && !seen.Has(p) {
			paths[kept] = p
			kept++
			seen.Insert(p)
		}
	}
	return paths[:kept]
}

// homeDir returns the home directory for the current user.
// On Windows:
// 1. the first of %HOME%, %HOMEDRIVE%%HOMEPATH%, %USERPROFILE% containing a `.kube\config` file is returned.
// 2. if none of those locations contain a `.kube\config` file, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that exists and is writeable is returned.
// 3. if none of those locations are writeable, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that exists is returned.
// 4. if none of those locations exists, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that is set is returned.
// NOTE this is from client-go. Rather than pull in client-go for this one
// standalone method, we have a fork here.
// https://github.com/kubernetes/client-go/blob/6d7018244d72350e2e8c4a19ccdbe4c8083a9143/util/homedir/homedir.go
// We've modified this to require injecting os.Getenv and runtime.GOOS as a dependencies for testing purposes
func homeDir(GOOS string, getEnv func(string) string) string {
	if GOOS == "windows" {
		home := getEnv("HOME")
		homeDriveHomePath := ""
		if homeDrive, homePath := getEnv("HOMEDRIVE"), getEnv("HOMEPATH"); len(homeDrive) > 0 && len(homePath) > 0 {
			homeDriveHomePath = homeDrive + homePath
		}
		userProfile := getEnv("USERPROFILE")

		// Return first of %HOME%, %HOMEDRIVE%/%HOMEPATH%, %USERPROFILE% that contains a `.kube\config` file.
		// %HOMEDRIVE%/%HOMEPATH% is preferred over %USERPROFILE% for backwards-compatibility.
		for _, p := range []string{home, homeDriveHomePath, userProfile} {
			if len(p) == 0 {
				continue
			}
			if _, err := os.Stat(filepath.Join(p, ".kube", "config")); err != nil {
				continue
			}
			return p
		}

		firstSetPath := ""
		firstExistingPath := ""

		// Prefer %USERPROFILE% over %HOMEDRIVE%/%HOMEPATH% for compatibility with other auth-writing tools
		for _, p := range []string{home, userProfile, homeDriveHomePath} {
			if len(p) == 0 {
				continue
			}
			if len(firstSetPath) == 0 {
				// remember the first path that is set
				firstSetPath = p
			}
			info, err := os.Stat(p)
			if err != nil {
				continue
			}
			if len(firstExistingPath) == 0 {
				// remember the first path that exists
				firstExistingPath = p
			}
			if info.IsDir() && info.Mode().Perm()&(1<<(uint(7))) != 0 {
				// return first path that is writeable
				return p
			}
		}

		// If none are writeable, return first location that exists
		if len(firstExistingPath) > 0 {
			return firstExistingPath
		}

		// If none exist, return first location that is set
		if len(firstSetPath) > 0 {
			return firstSetPath
		}

		// We've got nothing
		return ""
	}
	return getEnv("HOME")
}
