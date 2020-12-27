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

package yaml

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMarshalWithSeparator(t *testing.T) {
	type Data struct {
		Foo string `yaml:"foo"`
	}

	tests := []struct {
		description string
		input       []Data
		expected    string
	}{
		{
			description: "single element slice",
			input: []Data{
				{Foo: "foo"},
			},
			expected: "foo: foo\n",
		},
		{
			description: "multi element slice",
			input: []Data{
				{Foo: "foo1"},
				{Foo: "foo2"},
			},
			expected: "foo: foo1\n---\nfoo: foo2\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			output, err := MarshalWithSeparator(test.input)
			t.CheckNoError(err)
			t.CheckDeepEqual(string(output), test.expected)
		})
	}
}
