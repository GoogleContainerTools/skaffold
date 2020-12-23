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

package analyze

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

type skaffoldConfigAnalyzer struct {
	directoryAnalyzer
	force        bool
	analyzeMode  bool
	targetConfig string
}

func (a *skaffoldConfigAnalyzer) analyzeFile(filePath string) error {
	if !schema.IsSkaffoldConfig(filePath) || a.force || a.analyzeMode {
		return nil
	}
	sameFiles, err := sameFiles(filePath, a.targetConfig)
	if err != nil {
		return fmt.Errorf("failed to analyze file %s: %s", filePath, err)
	}
	if !sameFiles {
		return nil
	}

	return errors.PreExistingConfigErr{Path: filePath}
}

func sameFiles(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return absA == absB, nil
}
