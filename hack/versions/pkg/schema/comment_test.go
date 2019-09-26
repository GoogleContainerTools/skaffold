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

package schema

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

const configFileTemplate = `/*
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

package v1beta12

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

%sconst Version string = "skaffold/v1beta12"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}
`

var configWithNoComment = fmt.Sprintf(configFileTemplate, "")
var configWithReleasedComment = fmt.Sprintf(configFileTemplate, releasedComment+"\n")
var configWithUnreleasedComment = fmt.Sprintf(configFileTemplate, unreleasedComment+"\n")

func TestUpdateComments(t *testing.T) {

	tcs := []struct {
		name     string
		orig     string
		expected string
		released bool
	}{
		{
			name:     "unreleased comment added on file",
			released: true,
			orig:     configWithNoComment,
			expected: configWithReleasedComment,
		},
		{
			name:     "released comment added on file",
			released: false,
			orig:     configWithNoComment,
			expected: configWithUnreleasedComment,
		},
		{
			name:     "released -> released",
			released: true,
			orig:     configWithReleasedComment,
			expected: configWithReleasedComment,
		},
		{
			name:     "unreleased -> unreleased",
			released: false,
			orig:     configWithUnreleasedComment,
			expected: configWithUnreleasedComment,
		},
		{
			name:     "released -> unreleased",
			released: false,
			orig:     configWithReleasedComment,
			expected: configWithUnreleasedComment,
		},
		{
			name:     "unreleased -> released",
			released: true,
			orig:     configWithUnreleasedComment,
			expected: configWithReleasedComment,
		},
	}

	for _, tc := range tcs {
		testutil.Run(t, tc.name, func(t *testutil.T) {

			dir := t.NewTempDir()
			aFile := dir.Path("a.go")
			t.CheckNoError(ioutil.WriteFile(aFile, []byte(tc.orig), 0666))
			modified, err := updateVersionComment(aFile, tc.released)
			t.CheckErrorAndDeepEqual(false, err, tc.expected, string(modified))
		})
	}

}
