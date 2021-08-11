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
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
)

// CompareGoStructs returns an empty string iff aFile and bFile are valid go files
// and the top level go struct declarations have the same fields independent of order.
// It returns a semi-readable diff of the AST in case the two files are different.
// Returns error when either of the go files are not parseable.
func CompareGoStructs(aFile string, bFile string) (string, error) {
	fset := token.NewFileSet()
	astA, err := parser.ParseFile(fset, aFile, nil, parser.AllErrors)
	if err != nil {
		return "", err
	}
	astB, err := parser.ParseFile(fset, bFile, nil, parser.AllErrors)
	if err != nil {
		return "", err
	}

	return cmp.Diff(structsMap(astA), structsMap(astB)), nil
}

func structsMap(astB *ast.File) map[string]string {
	bStructs := make(map[string]string)
	for _, n := range astB.Decls {
		decl, ok := n.(*ast.GenDecl)
		if !ok {
			continue
		}
		typeSec, ok := decl.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := typeSec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		bStructs[typeSec.Name.Name] = fieldListString(structType)
	}
	logrus.Debugf("%+v", bStructs)
	return bStructs
}

func fieldListString(structType *ast.StructType) string {
	fieldListString := ""
	fields := structType.Fields.List
	sort.Slice(fields, func(i, j int) bool {
		if len(fields[i].Names) == 0 {
			return false
		}
		if len(fields[i].Names) > 0 && len(fields[j].Names) == 0 {
			return true
		}
		return strings.Compare(fields[i].Names[0].Name, fields[j].Names[0].Name) > 0
	})
	for _, field := range fields {
		tag := "'"
		if field.Tag != nil {
			tag = field.Tag.Value
		}
		fieldListString = fmt.Sprintf("%s %s type: %s tag: %s",
			fieldListString,
			field.Names,
			baseTypeName(field.Type),
			tag)
	}
	return fieldListString
}

// inspired by https://github.com/golang/go/blob/9b968df17782f21cc0af14c9d3c0bcf4cf3f911f/src/go/doc/reader.go#L100
func baseTypeName(x ast.Expr) (name string) {
	switch t := x.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if _, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name
		}
	case *ast.ArrayType:
		return "[]" + baseTypeName(t.Elt)
	case *ast.MapType:
		return "map[" + baseTypeName(t.Key) + "]" + baseTypeName(t.Value)
	case *ast.ParenExpr:
		return baseTypeName(t.X)
	case *ast.StarExpr:
		return "*" + baseTypeName(t.X)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		panic(fmt.Errorf("not covered %+v %+v ", t, x))
	}
	return
}
