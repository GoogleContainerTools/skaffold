/*
Copyright 2021 The Skaffold Authors

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

package jib

import (
	"os/exec"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	// JVMFound is true if a Java VM was found and works.
	resolveJVMOnce sync.Once
	jvmPresent bool
)

// JVMFound returns true if a Java VM was found and works.
func JVMFound() bool {
	resolveJVMOnce.Do(func() {
		jvmPresent = resolveJVM()
	})
	return jvmPresent
}

// resolveJVMForInit returns true if a Java VM was found and works.  It is intended for
// `skaffold init` on macOS where calling out to the Maven Wrapper script (mvnw) can
// hang if there is no installed Java VM found.
func resolveJVM() bool {
	// TODO: should we have an override for testing?
	cmd := exec.Command("java", "-version")
	err := util.RunCmd(cmd)
	if err != nil {
		logrus.Warnf("Skipping Jib: no JVM: %v failed: %v", cmd.Args, err)
	}
	return err == nil
}
