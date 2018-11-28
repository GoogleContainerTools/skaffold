/*
   Copyright The containerd Authors.

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

package continuity

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
)

// TestUniqifyDigests ensures that we deterministically sort digest entries
// for a resource.
func TestUniqifyDigests(t *testing.T) {
	for _, testcase := range []struct {
		description string
		input       [][]digest.Digest // input is a series of digest collections from separate resources.
		expected    []digest.Digest
		err         error
	}{
		{
			description: "simple merge",
			input: [][]digest.Digest{
				{"sha1:abc", "sha256:def"},
				{"sha1:abc", "sha256:def"},
			},
			expected: []digest.Digest{"sha1:abc", "sha256:def"},
		},
		{
			description: "simple reversed order",
			input: [][]digest.Digest{
				{"sha1:abc", "sha256:def"},
				{"sha256:def", "sha1:abc"},
			},
			expected: []digest.Digest{"sha1:abc", "sha256:def"},
		},
		{
			description: "conflicting values for sha1",
			input: [][]digest.Digest{
				{"sha1:abc", "sha256:def"},
				{"sha256:def", "sha1:def"},
			},
			err: fmt.Errorf("conflicting digests for sha1 found"),
		},
	} {
		fatalf := func(format string, args ...interface{}) {
			t.Fatalf(testcase.description+": "+format, args...)
		}

		var assembled []digest.Digest

		for _, ds := range testcase.input {
			assembled = append(assembled, ds...)
		}

		merged, err := uniqifyDigests(assembled...)
		if err != testcase.err {
			if testcase.err == nil {
				fatalf("unexpected error uniqifying digests: %v", err)
			}

			if err != testcase.err && err.Error() != testcase.err.Error() {
				// compare by string till we create nice error type
				fatalf("unexpected error uniqifying digests: %v != %v", err, testcase.err)
			}
		}

		if !reflect.DeepEqual(merged, testcase.expected) {
			fatalf("unexpected uniquification: %v != %v", merged, testcase.expected)
		}

	}
}
