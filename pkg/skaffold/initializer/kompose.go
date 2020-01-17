/*
Copyright 2020 The Skaffold Authors

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

package initializer

import (
	"context"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// runKompose runs the `kompose` CLI before running skaffold init
func runKompose(ctx context.Context, composeFile string) error {
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return err
	}

	logrus.Infof("running 'kompose convert' for file %s", composeFile)
	komposeCmd := exec.CommandContext(ctx, "kompose", "convert", "-f", composeFile)
	_, err := util.RunCmdOut(komposeCmd)
	return err
}
