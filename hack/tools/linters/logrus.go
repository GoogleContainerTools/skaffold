/*
Copyright 2021 The Skaffold Authors

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

package linters

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var LogrusAnalyzer = &analysis.Analyzer{
	Name: "logruslinter",
	Doc:  "find usage of logrus",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if importSpec, ok := n.(*ast.ImportSpec); ok {
				if importSpec.Path != nil && strings.Contains(importSpec.Path.Value, "github.com/sirupsen/logrus")	{
					pass.Report(analysis.Diagnostic{
						Pos:            importSpec.Pos(),
						End:            0,
						Category:       "logrus-analyzer",
						Message:        "Dont use github.com/sirupsen/logrus package, use output.Log instead.",
					})
				}
			}
			return true
		})
	}
	return nil, nil
}
