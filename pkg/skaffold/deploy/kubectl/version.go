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

package kubectl

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const unknown = "unknown"

// Version is the client version of kubectl.
type Version struct {
	Client ClientVersion `json:"clientVersion"`
}

// ClientVersion is the client version of kubectl.
type ClientVersion struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
}

func (v ClientVersion) String() string {
	if v.Major == unknown || v.Minor == unknown {
		return unknown
	}

	return v.Major + "." + v.Minor
}

// CheckVersion warns the user if their kubectl version is < 1.12.0
func (c *CLI) CheckVersion(ctx context.Context) error {
	m, err := strconv.Atoi(c.Version(ctx).Minor)
	if err != nil {
		return errors.Wrap(err, "couldn't get kubectl minor version")
	}
	if m < 12 {
		return errors.New("kubectl version 1.12.0 or greater is recommended for use with Skaffold")
	}
	return nil
}

// Version returns the client version of kubectl.
func (c *CLI) Version(ctx context.Context) ClientVersion {
	c.versionOnce.Do(func() {
		version := Version{
			Client: ClientVersion{
				Major: unknown,
				Minor: unknown,
			},
		}

		buf, err := c.getVersion(ctx)
		if err != nil {
			logrus.Warnln("unable to get kubectl client version", err)
		} else if err := json.Unmarshal(buf, &version); err != nil {
			logrus.Warnln("unable to parse client version", err)
		}

		c.version = version.Client
	})

	return c.version
}

func (c *CLI) getVersion(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "version", "--client", "-ojson")
	return util.RunCmdOut(cmd)
}
