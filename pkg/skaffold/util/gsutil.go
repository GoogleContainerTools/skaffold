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

package util

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const GsutilExec = "gsutil"

type Gsutil struct{}

// Copy calls `gsutil cp [-r] <source_url> <destination_url>
func (g *Gsutil) Copy(ctx context.Context, src, dst string, recursive bool) error {
	args := []string{"cp"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, src, dst)
	cmd := exec.CommandContext(ctx, GsutilExec, args...)
	out, err := RunCmdOut(cmd)
	if err != nil {
		return fmt.Errorf("copy file(s) with %s failed: %w", GsutilExec, err)
	}
	logrus.Info(out)
	return nil
}
