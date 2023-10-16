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

package docker

import (
	"encoding/json"
	"strings"

	"sigs.k8s.io/kind/pkg/exec"
)

// IsAvailable checks if docker is available in the system
func IsAvailable() bool {
	cmd := exec.Command("docker", "-v")
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return false
	}
	return strings.HasPrefix(lines[0], "Docker version")
}

// usernsRemap checks if userns-remap is enabled in dockerd
func usernsRemap() bool {
	cmd := exec.Command("docker", "info", "--format", "'{{json .SecurityOptions}}'")
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return false
	}
	if len(lines) > 0 {
		if strings.Contains(lines[0], "name=userns") {
			return true
		}
	}
	return false
}

// mountDevMapper checks if the Docker storage driver is Btrfs or ZFS
// or if the backing filesystem is Btrfs
func mountDevMapper() bool {
	storage := ""
	// check the docker storage driver
	cmd := exec.Command("docker", "info", "-f", "{{.Driver}}")
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return false
	}

	storage = strings.ToLower(strings.TrimSpace(lines[0]))
	if storage == "btrfs" || storage == "zfs" || storage == "devicemapper" {
		return true
	}

	// check the backing file system
	// docker info -f '{{json .DriverStatus  }}'
	// [["Backing Filesystem","extfs"],["Supports d_type","true"],["Native Overlay Diff","true"]]
	cmd = exec.Command("docker", "info", "-f", "{{json .DriverStatus }}")
	lines, err = exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return false
	}
	var dat [][]string
	if err := json.Unmarshal([]byte(lines[0]), &dat); err != nil {
		return false
	}
	for _, item := range dat {
		if item[0] == "Backing Filesystem" {
			storage = strings.ToLower(item[1])
			break
		}
	}

	return storage == "btrfs" || storage == "zfs" || storage == "xfs"
}

// rootless: use fuse-overlayfs by default
// https://github.com/kubernetes-sigs/kind/issues/2275
func mountFuse() bool {
	i, err := info()
	if err != nil {
		return false
	}
	if i != nil && i.Rootless {
		return true
	}
	return false
}
