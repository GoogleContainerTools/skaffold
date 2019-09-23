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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"

	"github.com/sirupsen/logrus"
)

func TestCmpGoStructs(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	tcs := []struct {
		name      string
		a         string
		b         string
		same      bool
		shouldErr bool
	}{
		{
			name:      "same strings",
			a:         `package a`,
			b:         `package a`,
			same:      true,
			shouldErr: false,
		},
		{
			name:      "invalid go file",
			a:         `package a`,
			b:         `invalid`,
			same:      true,
			shouldErr: true,
		},
		{
			name: "comment changes: same",
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
			name: "all supported types",
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
			name: "renamed struct: not same",
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
			name: "added struct: not same",
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
			name: "removed struct: not same",
			a: `package a
type TestStructure struct {} 
`,
			b: `package a
`,
			same:      false,
			shouldErr: false,
		},
		{
			name: "added field: not same",
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
			name: "renamed field: not same",
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
			name: "type change of field: not same",
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
			name: "type change of field pointer: not same",
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
			name: "reordered fields: same",
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
			name: "reordered structs: same",
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
			name: "change in yaml tag: not same",
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

	for _, tc := range tcs {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			dir := t.NewTempDir()
			aFile := dir.Path("a.go")
			bFile := dir.Path("b.go")
			t.CheckNoError(ioutil.WriteFile(aFile, []byte(tc.a), 0666))
			t.CheckNoError(ioutil.WriteFile(bFile, []byte(tc.b), 0666))
			diff, err := CompareGoStructs(aFile, bFile)
			t.CheckErrorAndDeepEqual(tc.shouldErr, err, tc.same, diff == "")
			if tc.same != (diff == "") {
				t.Errorf(diff)
			}
		})
	}
}

func TestCompareSchemas(t *testing.T) {
	tcs := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "should be same",
			a:    "v1beta13",
			b:    "v1beta13",
		},
		{
			name: "should be different",
			a:    "v1beta12",
			b:    "v1beta13",
		},
	}
	logrus.SetLevel(logrus.DebugLevel)
	for _, tc := range tcs {
		testutil.Run(t, tc.name, func(t *testutil.T) {
			wd, err := os.Getwd()
			t.CheckNoError(err)
			slashWd := filepath.ToSlash(wd)
			schemaDir := path.Join(slashWd, "..", "..", "..", "..", "pkg", "skaffold", "schema")
			a := path.Join(schemaDir, tc.a, "config.go")
			b := path.Join(schemaDir, tc.b, "config.go")
			diff, err := CompareGoStructs(filepath.FromSlash(a), filepath.FromSlash(b))
			t.CheckErrorAndDeepEqual(false, err, tc.a == tc.b, diff == "")
			if diff != "" && tc.a == tc.b {
				t.Errorf(diff)
			}
		})
	}

}
