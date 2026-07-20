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

// Ported from https://github.com/genuinetools/bpfd/blob/a4bfa5e3e9d1bfdbc56268a36a0714911ae9b6bf/proc/proc.go

package proc

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ContainerRuntime is the type for the various container runtime strings.
type ContainerRuntime string

const (
	// RuntimeDocker is the string for the docker runtime.
	RuntimeDocker ContainerRuntime = "docker"
	// RuntimeRkt is the string for the rkt runtime.
	RuntimeRkt ContainerRuntime = "rkt"
	// RuntimeNspawn is the string for the systemd-nspawn runtime.
	RuntimeNspawn ContainerRuntime = "systemd-nspawn"
	// RuntimeLXC is the string for the lxc runtime.
	RuntimeLXC ContainerRuntime = "lxc"
	// RuntimeLXCLibvirt is the string for the lxc-libvirt runtime.
	RuntimeLXCLibvirt ContainerRuntime = "lxc-libvirt"
	// RuntimeOpenVZ is the string for the openvz runtime.
	RuntimeOpenVZ ContainerRuntime = "openvz"
	// RuntimeKubernetes is the string for the kubernetes runtime.
	RuntimeKubernetes ContainerRuntime = "kube"
	// RuntimeGarden is the string for the garden runtime.
	RuntimeGarden ContainerRuntime = "garden"
	// RuntimePodman is the string for the podman runtime.
	RuntimePodman ContainerRuntime = "podman"
	// RuntimeGVisor is the string for the gVisor (runsc) runtime.
	RuntimeGVisor ContainerRuntime = "gvisor"
	// RuntimeFirejail is the string for the firejail runtime.
	RuntimeFirejail ContainerRuntime = "firejail"
	// RuntimeWSL is the string for the Windows Subsystem for Linux runtime.
	RuntimeWSL ContainerRuntime = "wsl"
	// RuntimeNotFound is the string for when no container runtime is found.
	RuntimeNotFound ContainerRuntime = "not-found"
)

var (
	// ContainerRuntimes contains all the container runtimes.
	ContainerRuntimes = []ContainerRuntime{
		RuntimeDocker,
		RuntimeRkt,
		RuntimeNspawn,
		RuntimeLXC,
		RuntimeLXCLibvirt,
		RuntimeOpenVZ,
		RuntimeKubernetes,
		RuntimeGarden,
		RuntimePodman,
		RuntimeGVisor,
		RuntimeFirejail,
		RuntimeWSL,
	}
)

// GetContainerRuntime returns the container runtime the process is running in.
// If pid is less than one, it returns the runtime for "self".
func GetContainerRuntime(tgid, pid int) ContainerRuntime {
	file := "/proc/self/cgroup"
	if pid > 0 {
		if tgid > 0 {
			file = fmt.Sprintf("/proc/%d/task/%d/cgroup", tgid, pid)
		} else {
			file = fmt.Sprintf("/proc/%d/cgroup", pid)
		}
	}

	// read the cgroups file
	a := readFileString(file)
	runtime := getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// /proc/vz exists in container and outside of the container, /proc/bc only outside of the container.
	if osFileExists("/proc/vz") && !osFileExists("/proc/bc") {
		return RuntimeOpenVZ
	}

	// /__runsc_containers__ directory is present in gVisor containers.
	if osFileExists("/__runsc_containers__") {
		return RuntimeGVisor
	}

	// firejail runs with `firejail` as pid 1.
	// As firejail binary cannot be run with argv[0] != "firejail"
	// it's okay to rely on cmdline.
	a = readFileString("/proc/1/cmdline")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// WSL has /proc/version_signature starting with "Microsoft".
	a = readFileString("/proc/version_signature")
	if strings.HasPrefix(a, "Microsoft") {
		return RuntimeWSL
	}

	a = os.Getenv("container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// PID 1 might have dropped this information into a file in /run.
	// Read from /run/systemd/container since it is better than accessing /proc/1/environ,
	// which needs CAP_SYS_PTRACE
	a = readFileString("/run/systemd/container")
	runtime = getContainerRuntime(a)
	if runtime != RuntimeNotFound {
		return runtime
	}

	// Check for container specific files
	runtime = detectContainerFiles()
	if runtime != RuntimeNotFound {
		return runtime
	}

	// Docker was not detected at this point.
	// An overlay mount on "/" may indicate we're under containerd or other runtime.
	a = readFileString("/proc/mounts")
	if m, _ := regexp.MatchString("^[^ ]+ / overlay", a); m {
		return RuntimeKubernetes
	}

	return RuntimeNotFound
}

// Related implementation: https://github.com/systemd/systemd/blob/6604fb0207ee10e8dc05d67f6fe45de0b193b5c4/src/basic/virt.c#L523-L549
func detectContainerFiles() ContainerRuntime {
	files := []struct {
		runtime  ContainerRuntime
		location string
	}{
		// https://github.com/containers/podman/issues/6192
		// https://github.com/containers/podman/issues/3586#issuecomment-661918679
		{RuntimePodman, "/run/.containerenv"},
		// https://github.com/moby/moby/issues/18355
		{RuntimeDocker, "/.dockerenv"},
		// Detect the presence of a serviceaccount secret mounted in the default location
		{RuntimeKubernetes, "/var/run/secrets/kubernetes.io/serviceaccount"},
	}

	for i := range files {
		if osFileExists(files[i].location) {
			return files[i].runtime
		}
	}

	return RuntimeNotFound
}

func getContainerRuntime(input string) ContainerRuntime {
	if len(strings.TrimSpace(input)) < 1 {
		return RuntimeNotFound
	}

	for _, runtime := range ContainerRuntimes {
		if strings.Contains(input, string(runtime)) {
			return runtime
		}
	}

	return RuntimeNotFound
}

func osFileExists(file string) bool {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		return true
	}
	return false
}

func readFile(file string) []byte {
	if !osFileExists(file) {
		return nil
	}

	b, _ := os.ReadFile(file)
	return b
}

func readFileString(file string) string {
	b := readFile(file)
	if b == nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
