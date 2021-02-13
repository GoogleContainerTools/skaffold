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

package structure

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Dependencies returns dependencies listed for the structure tests
func (tr *Runner) TestDependencies() ([]string, error) {
	files, err := util.ExpandPathsGlob(tr.testWorkingDir, tr.structureTests)
	if err != nil {
		return nil, expandingFilePathsErr(err)
	}

	return files, nil
}
