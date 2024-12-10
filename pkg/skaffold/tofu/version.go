/*
Copyright 2019 The Skaffold Authors

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

package tofu

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
)

const unknown = "unknown"

// Version is the version of tofu.
type Version struct {
	TerraformVersion string `json:"terraform_version"`
}

type ClientVersion string

// Version returns the client version of tofu.
func (c *CLI) Version(ctx context.Context) ClientVersion {
	c.versionOnce.Do(func() {
		version := Version{
			TerraformVersion: unknown,
		}

		buf, err := c.getVersion(ctx)
		if err != nil {
			warnings.Printf("unable to get tofu version: %v", err)
		} else if err := json.Unmarshal(buf, &version); err != nil {
			warnings.Printf("unable to parse version: %v", err)
		}

		c.version = ClientVersion(version.TerraformVersion)
	})

	return c.version
}

func (c *CLI) getVersion(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tofu", "version", "-json")
	return util.RunCmdOut(ctx, cmd)
}
