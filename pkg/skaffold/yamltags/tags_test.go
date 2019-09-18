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

package yamltags

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

type otherstruct struct {
	A int `yamltags:"required"`
}

type required struct {
	A string `yamltags:"required"`
	B int    `yamltags:"required"`
	C otherstruct
}

func TestValidateStructRequired(t *testing.T) {
	type args struct {
		s interface{}
	}

	tests := []struct {
		description string
		args        args
		shouldErr   bool
	}{
		{
			description: "missing all",
			args: args{
				s: &required{},
			},
			shouldErr: true,
		},
		{
			description: "all set",
			args: args{
				s: &required{
					A: "hey",
					B: 3,
					C: otherstruct{
						A: 1,
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "missing some",
			args: args{
				s: &required{
					A: "hey",
					C: otherstruct{
						A: 1,
					},
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := ValidateStruct(test.args.s)

			t.CheckError(test.shouldErr, err)
		})
	}
}

type oneOfStruct struct {
	A string  `yamltags:"oneOf=set1"`
	B string  `yamltags:"oneOf=set1"`
	C int     `yamltags:"oneOf=set2"`
	D *nested `yamltags:"oneOf=set2"`
	E nested  `yamltags:"oneOf=set2"`
}

type nested struct {
	F string
}

func TestValidateStructOneOf(t *testing.T) {
	type args struct {
		s interface{}
	}

	tests := []struct {
		description string
		args        args
		shouldErr   bool
	}{
		{
			description: "only one",
			args: args{
				s: &oneOfStruct{
					A: "foo",
					C: 3,
				},
			},
			shouldErr: false,
		},
		{
			description: "too many in one set",
			args: args{
				s: &oneOfStruct{
					A: "foo",
					B: "baz",
					C: 3,
				}},
			shouldErr: true,
		},
		{
			description: "too many pointers set",
			args: args{
				s: &oneOfStruct{
					D: &nested{F: "foo"},
					E: nested{F: "foo"},
				}},
			shouldErr: true,
		},
		{
			description: "too many in both sets",
			args: args{
				s: &oneOfStruct{
					A: "foo",
					B: "baz",
					C: 3,
					D: &nested{F: "foo"},
				}},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			err := ValidateStruct(test.args.s)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestValidateStructInvalid(t *testing.T) {
	defer testutil.EnsureTestPanicked(t)

	invalidTags := struct {
		A string `yamltags:"invalidTag"`
	}{}

	ValidateStruct(invalidTags)
}

func TestValidateStructInvalidSyntax(t *testing.T) {
	invalidTags := struct {
		A string `yamltags:"oneOf=set1=set2"`
	}{}

	err := ValidateStruct(invalidTags)

	testutil.CheckError(t, true, err)
}

func TestIsZeroValue(t *testing.T) {
	testutil.CheckDeepEqual(t, true, isZeroValue(reflect.ValueOf(0)))
	testutil.CheckDeepEqual(t, true, isZeroValue(reflect.ValueOf(nil)))
	var zeroMap map[string]string
	testutil.CheckDeepEqual(t, true, isZeroValue(reflect.ValueOf(zeroMap)))

	nonZeroMap := make(map[string]string)
	testutil.CheckDeepEqual(t, false, isZeroValue(reflect.ValueOf(nonZeroMap)))
}

func TestYamlName(t *testing.T) {
	object := struct {
		Empty   string `yaml:",omitempty"`
		Named   string `yaml:"named,omitempty"`
		Missing string
	}{}
	testutil.CheckDeepEqual(t, "Empty", YamlName(reflect.TypeOf(object).Field(0)))
	testutil.CheckDeepEqual(t, "named", YamlName(reflect.TypeOf(object).Field(1)))
	testutil.CheckDeepEqual(t, "Missing", YamlName(reflect.TypeOf(object).Field(2)))
}
