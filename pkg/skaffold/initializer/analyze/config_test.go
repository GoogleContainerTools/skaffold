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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestConfigAnalyzer(t *testing.T) {
	tests := []struct {
		name      string
		inputFile string
		analyzer  skaffoldConfigAnalyzer
		shouldErr bool
	}{
		{
			name:      "not skaffold config",
			inputFile: "../testdata/init/hello/main.go",
			analyzer:  skaffoldConfigAnalyzer{},
			shouldErr: false,
		},
		{
			name:      "skaffold config equals target config",
			inputFile: "../testdata/init/hello/skaffold.yaml",
			analyzer: skaffoldConfigAnalyzer{
				targetConfig: "../testdata/init/hello/skaffold.yaml",
			},
			shouldErr: true,
		},
		{
			name:      "skaffold config does not equal target config",
			inputFile: "../testdata/init/hello/skaffold.yaml",
			analyzer: skaffoldConfigAnalyzer{
				targetConfig: "../testdata/init/hello/skaffold.yaml.out",
			},
			shouldErr: false,
		},
		{
			name:      "force overrides",
			inputFile: "../testdata/init/hello/skaffold.yaml",
			analyzer: skaffoldConfigAnalyzer{
				force:        true,
				targetConfig: "../testdata/init/hello/skaffold.yaml",
			},
			shouldErr: false,
		},
		{
			name:      "analyze mode can skip writing, no error",
			inputFile: "../testdata/init/hello/skaffold.yaml",
			analyzer: skaffoldConfigAnalyzer{
				force:        false,
				analyzeMode:  true,
				targetConfig: testutil.Abs(t, "../testdata/init/hello/skaffold.yaml"),
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			err := test.analyzer.analyzeFile(test.inputFile)
			t.CheckError(test.shouldErr, err)
		})
	}
}
