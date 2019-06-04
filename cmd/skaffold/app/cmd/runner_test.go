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

package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func writeSkaffoldConfig(t *testing.T, content string) (string, func()) {
	yaml := fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latest.Version, content)
	return testutil.TempFile(t, "skaffold.yaml", []byte(yaml))
}

func TestNewRunner(t *testing.T) {
	cfg, delete := writeSkaffoldConfig(t, "")
	defer delete()

	_, _, err := newRunner(&config.SkaffoldOptions{
		ConfigurationFile: cfg,
		BuildTrigger:      "polling",
	})

	testutil.CheckError(t, false, err)
}

func TestNewRunnerMissingConfig(t *testing.T) {
	_, _, err := newRunner(&config.SkaffoldOptions{
		ConfigurationFile: "missing-skaffold.yaml",
	})

	testutil.CheckError(t, true, err)
	if !os.IsNotExist(errors.Cause(err)) {
		t.Error("error should say that file is missing")
	}
}

func TestNewRunnerInvalidConfig(t *testing.T) {
	cfg, delete := writeSkaffoldConfig(t, "invalid")
	defer delete()

	_, _, err := newRunner(&config.SkaffoldOptions{
		ConfigurationFile: cfg,
	})

	testutil.CheckErrorContains(t, "parsing skaffold config", err)
}

func TestNewRunnerUnknownProfile(t *testing.T) {
	cfg, delete := writeSkaffoldConfig(t, "")
	defer delete()

	_, _, err := newRunner(&config.SkaffoldOptions{
		ConfigurationFile: cfg,
		Profiles:          []string{"unknown-profile"},
	})

	testutil.CheckErrorContains(t, "applying profiles", err)
}
