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

package integration

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
)

func TestDiagnose(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	out, err := skaffold.Diagnose("-vdebug").InDir(filepath.Join("examples", "jib-gradle")).RunWithCombinedOutput(t)
	failNowIfError(t, err)
	t.Log(string(out))
}

func folders(root string) ([]string, error) {
	var folders []string

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.Mode().IsDir() {
			folders = append(folders, f.Name())
		}
	}

	return folders, err
}
