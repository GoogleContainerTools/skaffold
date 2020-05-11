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

package diff

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCmpGoStructs(t *testing.T) {
	tests := []struct {
		description string
		a           string
		b           string
		same        bool
		shouldErr   bool
	}{
		{
			description: "same strings",
			a:           `package a`,
			b:           `package a`,
			same:        true,
			shouldErr:   false,
		},
		{
			description: "invalid go file",
			a:           `package a`,
			b:           `invalid`,
			same:        true,
			shouldErr:   true,
		},
		{
			description: "comment changes: same",
			a: `package a
//a comment
`,
			b: `package a
//a different comment
`,
			same:      true,
			shouldErr: false,
		},
		{
			description: "all supported types",
			a: `package a
type TestStructure struct {
	a string			// Ident
	b map[string]int 	// MapType
	c []interface{} 	// InterfaceType
	d []byte	 		// ArrayType
	e (error)    		// ParenExpr
	f *ast.Ident 		// SelectorExpr and StarExpr
}
`,
			b: `package a
type TestStructure struct {
	a string			// Ident
	b map[string]int 	// MapType
	c []interface{} 	// InterfaceType
	d []byte	 		// ArrayType
	e (error)    		// ParenExpr
	f *ast.Ident 		// SelectorExpr and StarExpr
}
`,
			same:      true,
			shouldErr: false,
		},
		{
			description: "renamed struct: not same",
			a: `package a
//a comment
type TestStructure struct {} 
`,
			b: `package a
//a different comment
type TestStructureRenamed struct {} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "added struct: not same",
			a: `package a
//a comment
type TestStructure struct {} 
`,
			b: `package a
//a different comment
type TestStructure struct {} 
type NewStructure struct {} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "removed struct: not same",
			a: `package a
type TestStructure struct {} 
`,
			b: `package a
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "added field: not same",
			a: `package a
type TestStructure struct {}
`,
			b: `package a
type TestStructure struct {
	newField string
} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "renamed field: not same",
			a: `package a
type TestStructure struct {
	oldField string
}
`,
			b: `package a
type TestStructure struct {
	newField string
} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "type change of field: not same",
			a: `package a
type TestStructure struct {
	oldField string
}
`,
			b: `package a
type TestStructure struct {
	oldField int
} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "type change of field pointer: not same",
			a: `package a
type TestStructure struct {
	oldField string
}
`,
			b: `package a
type TestStructure struct {
	oldField *string
} 
`,
			same:      false,
			shouldErr: false,
		},
		{
			description: "reordered fields: same",
			a: `package a
type TestStructure struct {
	fieldA string
	fieldB string
}
`,
			b: `package a
type TestStructure struct {
	fieldB string
	fieldA string
} 
`,
			same:      true,
			shouldErr: false,
		},
		{
			description: "reordered structs: same",
			a: `package a
type TestStructure struct {
	fieldA string
	fieldB string
}
type TestStructureB struct {
}
`,
			b: `package a
type TestStructureB struct {
}

type TestStructure struct {
	fieldB string
	fieldA string
}
`,
			same:      true,
			shouldErr: false,
		},
		{
			description: "change in yaml tag: not same",
			a: fmt.Sprintf(`package a
type TestStructure struct {
	fieldB string %s
	fieldA string
}
`, "`yaml:\"local\" yamltags:\"oneOf=build\"`"),
			b: fmt.Sprintf(`package a
type TestStructure struct {
	fieldB string %s
	fieldA string
}
`, "`yaml:\"local,omitempty\" yamltags:\"oneOf=build\"`"),
			same:      false,
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Write("a.go", test.a).
				Write("b.go", test.b).
				Chdir()

			diff, err := CompareGoStructs("a.go", "b.go")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.same, diff == "")
			if test.same != (diff == "") {
				t.Errorf(diff)
			}
		})
	}
}

func TestCompareSchemas(t *testing.T) {
	tests := []struct {
		description string
		a           string
		b           string
	}{
		{
			description: "should be same",
			a:           "v1beta13",
			b:           "v1beta13",
		},
		{
			description: "should be different",
			a:           "v1beta12",
			b:           "v1beta13",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			wd, err := os.Getwd()
			t.CheckNoError(err)
			slashWd := filepath.ToSlash(wd)
			schemaDir := path.Join(slashWd, "..", "..", "..", "..", "pkg", "skaffold", "schema")
			a := path.Join(schemaDir, test.a, "config.go")
			b := path.Join(schemaDir, test.b, "config.go")
			diff, err := CompareGoStructs(filepath.FromSlash(a), filepath.FromSlash(b))
			t.CheckNoError(err)
			t.CheckDeepEqual(test.a == test.b, diff == "")
			if diff != "" && test.a == test.b {
				t.Errorf(diff)
			}
		})
	}
}
