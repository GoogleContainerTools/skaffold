/*
Copyright 2018 The Skaffold Authors

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

package structure

import (
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"

	"github.com/sirupsen/logrus"
)

// Test is the entrypoint for running structure tests
func (tr *Runner) Test(image string) error {
	logrus.Infof("running structure tests for files %v", tr.testFiles)
	args := []string{"test", "--image", image}
	for _, f := range tr.testFiles {
		args = append(args, "--config", f)
	}
	args = append(args, tr.testFiles...)
	cmd := exec.Command("container-structure-test", args...)

	_, err := util.RunCmdOut(cmd)
	return err
}
