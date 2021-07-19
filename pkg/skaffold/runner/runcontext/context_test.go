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
package runcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	schemaUtil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetRunContextDefaultWorkdir(t *testing.T) {
	testutil.Run(t, "default workdir", func(t *testutil.T) {
		rctx, err := GetRunContext(config.SkaffoldOptions{}, []schemaUtil.VersionedConfig{})
		pwd, _ := os.Getwd()
		t.CheckDeepEqual(pwd, rctx.WorkingDir)
		t.CheckNoError(err)
	})
}

func TestGetRunContextCustomWorkdir(t *testing.T) {
	testutil.Run(t, "default workdir", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Write("skaffold.yaml", fmt.Sprintf("apiVersion: %s\nkind: Config", latestV1.Version)).
			Chdir()
		rctx, err := GetRunContext(config.SkaffoldOptions{
			ConfigurationFile: filepath.Join(tmpDir.Root(), "skaffold.yaml"),
		}, []schemaUtil.VersionedConfig{})
		t.CheckDeepEqual(tmpDir.Root(), rctx.WorkingDir)
		t.CheckNoError(err)
	})
}
