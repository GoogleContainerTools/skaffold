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
	"context"
	"os/exec"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	// JVMFound is replaceable for testing
	JVMFound = jvmFound

	// JVMFound() returns true if a Java VM was found and works.
	resolveJVMOnce sync.Once
	jvmPresent     bool
)

// jvmFound returns true if a Java VM was found and works.
func jvmFound(ctx context.Context) bool {
	// Check on demand: performing the check in an init() causes the
	// check to be run even when no jib functionality was used.
	resolveJVMOnce.Do(func() {
		jvmPresent = resolveJVM(ctx)
	})
	return jvmPresent
}

// resolveJVM returns true if a Java VM was found and works.  It is intended for
// `skaffold init` on macOS where calling out to the Maven Wrapper script (mvnw) can
// hang if there is no installed Java VM found.
func resolveJVM(ctx context.Context) bool {
	// Note that just checking for the existence of `java` is insufficient
	// as macOS ships with /usr/bin/java that tries to hand off to a JVM
	// installed in /Library/Java/JavaVirtualMachines
	cmd := exec.Command("java", "-version")
	err := util.RunCmd(ctx, cmd)
	if err != nil {
		log.Entry(context.TODO()).Warnf("Skipping Jib: no JVM: %v failed: %v", cmd.Args, err)
	}
	return err == nil
}
