/*
Copyright 2022 Google LLC

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

// Copied from https://github.com/osscontainertools/kaniko/blob/v1.28.3/pkg/util/proc/proc.go
// (GetContainerRuntime, itself ported from https://github.com/genuinetools/bpfd/blob/a4bfa5e3e9d1bfdbc56268a36a0714911ae9b6bf/proc/proc.go),
// reduced to the yes/no answer pack needs.

package container

import (
	"os"
	"regexp"
	"strings"
)

var containerRuntimes = []string{
	"docker",
	"rkt",
	"systemd-nspawn",
	"lxc",
	"lxc-libvirt",
	"openvz",
	"kube",
	"garden",
	"podman",
	"gvisor",
	"firejail",
	"wsl",
}

// RunningInContainer reports whether the current process runs inside a container.
func RunningInContainer() bool {
	for _, file := range []string{
		// https://github.com/containers/podman/issues/6192
		"/run/.containerenv",
		// https://github.com/moby/moby/issues/18355
		"/.dockerenv",
		"/var/run/secrets/kubernetes.io/serviceaccount",
		// gVisor
		"/__runsc_containers__",
	} {
		if fileExists(file) {
			return true
		}
	}

	// /proc/vz exists in container and outside of the container, /proc/bc only outside of the container.
	if fileExists("/proc/vz") && !fileExists("/proc/bc") {
		return true
	}

	// /proc/1/cmdline catches firejail, /run/systemd/container is where PID 1 may record $container.
	for _, s := range []string{
		readFileString("/proc/self/cgroup"),
		readFileString("/proc/1/cmdline"),
		os.Getenv("container"),
		readFileString("/run/systemd/container"),
	} {
		for _, runtime := range containerRuntimes {
			if strings.Contains(s, runtime) {
				return true
			}
		}
	}

	if strings.HasPrefix(readFileString("/proc/version_signature"), "Microsoft") {
		return true
	}

	// An overlay mount on "/" may indicate we're under containerd or other runtime.
	m, _ := regexp.MatchString("^[^ ]+ / overlay", readFileString("/proc/mounts"))
	return m
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func readFileString(file string) string {
	b, _ := os.ReadFile(file)
	return strings.TrimSpace(string(b))
}
