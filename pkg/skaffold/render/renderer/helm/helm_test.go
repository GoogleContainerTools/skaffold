/*
Copyright 2025 The Skaffold Authors

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

package helm

import (
	"errors"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestSkaffoldOverrides(t *testing.T) {
	testutil.Run(t, "no overrides present", func(t *testutil.T) {
		helmRelease := latest.HelmRelease{
			Overrides: util.HelmOverrides{Values: map[string]interface{}{}},
		}
		defer ensureHelmOverridesFileRemoved(t)

		deleteFn, err := generateSkaffoldOverrides(helmRelease)

		t.CheckNoError(err)
		t.CheckNil(deleteFn)
		t.CheckFileNotExist(constants.HelmOverridesFilename)
	})

	testutil.Run(t, "overrides present", func(t *testutil.T) {
		testFieldValue := "test-field-value"
		expectedContent := "test: " + testFieldValue + "\n"
		helmRelease := latest.HelmRelease{
			Overrides: util.HelmOverrides{Values: map[string]interface{}{
				"test": testFieldValue,
			}},
		}
		defer ensureHelmOverridesFileRemoved(t)

		deleteFn, err := generateSkaffoldOverrides(helmRelease)

		t.CheckNoError(err)
		t.CheckNotNil(deleteFn)
		t.CheckFileExistAndContent(constants.HelmOverridesFilename, []byte(expectedContent))
	})

	testutil.Run(t, "overrides present, delete handler functional", func(t *testutil.T) {
		helmRelease := latest.HelmRelease{
			Overrides: util.HelmOverrides{Values: map[string]interface{}{
				"test": []byte("test-content"),
			}},
		}
		defer ensureHelmOverridesFileRemoved(t)
		deleteFn, err := generateSkaffoldOverrides(helmRelease)
		defer t.CheckFileNotExist(constants.HelmOverridesFilename)

		defer deleteFn()

		t.CheckNoError(err)
		t.CheckNotNil(deleteFn)
		t.CheckFileExist(constants.HelmOverridesFilename)
	})

	testutil.Run(t, "error: file creation", func(t *testutil.T) {
		t.CheckNoError(os.Mkdir(constants.HelmOverridesFilename, 0o755))
		defer ensureHelmOverridesFileRemoved(t)
		helmRelease := latest.HelmRelease{
			Overrides: util.HelmOverrides{Values: map[string]interface{}{
				"test": "random-value",
			}},
		}

		deleteFn, err := generateSkaffoldOverrides(helmRelease)

		t.CheckErrorContains(`cannot create file "skaffold-overrides.yaml": open skaffold-overrides.yaml: is a directory`, err)
		t.CheckNil(deleteFn)
	})

	testutil.Run(t, "error: marshaling YAML", func(t *testutil.T) {
		helmRelease := latest.HelmRelease{
			Overrides: util.HelmOverrides{Values: map[string]interface{}{
				"failingContent": failingMarshalYAML{},
			}},
		}
		defer ensureHelmOverridesFileRemoved(t)

		deleteFn, err := generateSkaffoldOverrides(helmRelease)

		t.CheckErrorContains(failingYAMLMessage, err)
		t.CheckNil(deleteFn)
		t.CheckFileNotExist(constants.HelmOverridesFilename)
	})
}

func ensureHelmOverridesFileRemoved(t *testutil.T) {
	t.CheckNoError(os.RemoveAll(constants.HelmOverridesFilename))
}

var failingYAMLMessage = "failing marshal yaml"

type failingMarshalYAML struct{}

func (f failingMarshalYAML) MarshalYAML() (interface{}, error) {
	return nil, errors.New(failingYAMLMessage)
}
