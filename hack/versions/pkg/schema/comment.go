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
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
)

const releasedComment = `// !!! WARNING !!! This config version is already released, please DO NOT MODIFY the structs in this file.`
const unreleasedComment = `// This config version is not yet released, it is SAFE TO MODIFY the structs in this file.`

// recognizedComments is used to recognize whether an existing comment is a "release comment" or not.
// If you want to change releasedComment (or unreleasedComment) historically on all files, then:
// 1.) add the old version to `recognizedComments`
// 2.) change the text of `releasedComment` (or `unreleasedComment`)
// 3.) run `go run hack/versions/cmd/update_comments/main.go`
// 4.) remove the old version from `recognizedComments`
var recognizedComments = []string{
	releasedComment,
	unreleasedComment,
}

func UpdateVersionComment(origFile string, released bool) error {
	info, err := os.Stat(origFile)
	if err != nil {
		return err
	}
	content, err := updateVersionComment(origFile, released)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(origFile, content, info.Mode()); err != nil {
		return err
	}

	cmd := exec.Command("go", "fmt", origFile)
	return cmd.Run()
}

func updateVersionComment(origFile string, released bool) ([]byte, error) {
	fset := token.NewFileSet()

	var commentString string
	if released {
		commentString = releasedComment
	} else {
		commentString = unreleasedComment
	}

	astA, err := parser.ParseFile(fset, origFile, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	firstComment := astA.Comments[1].List[0]

	if firstComment.Text == commentString {
		return printAst(fset, astA)
	}

	if isRecognizedComment(firstComment) {
		firstComment.Text = commentString
		return printAst(fset, astA)
	}

	addFirstCommentOnVersion(astA, commentString)

	return printAst(fset, astA)
}

func isRecognizedComment(firstComment *ast.Comment) bool {
	for _, comment := range recognizedComments {
		if comment == firstComment.Text {
			return true
		}
	}
	return false
}

func printAst(fset *token.FileSet, ast *ast.File) ([]byte, error) {
	var buf bytes.Buffer

	if err := printer.Fprint(&buf, fset, ast); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addFirstCommentOnVersion(astA *ast.File, commentString string) {
	ast.Inspect(astA, func(node ast.Node) bool {
		if decl, ok := node.(*ast.GenDecl); ok &&
			decl.Tok.String() == "const" &&
			len(decl.Specs) == 1 {
			sp := decl.Specs[0]
			if t, ok := sp.(*ast.ValueSpec); ok {
				if len(t.Names) == 1 && t.Names[0].Name == "Version" {
					comment := ast.Comment{
						Slash: decl.TokPos - 1,
						Text:  commentString + "\n",
					}
					comments := []*ast.Comment{
						&comment,
					}

					cg := &ast.CommentGroup{List: comments}
					astA.Comments = append([]*ast.CommentGroup{astA.Comments[0], cg}, astA.Comments[1:]...)
					decl.Doc = cg
					return false
				}
			}
		}
		return true
	})
}
