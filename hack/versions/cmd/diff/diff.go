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

package main

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/diff"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
)

func main() {
	if len(os.Args) != 4 {
		color.Red.Fprintf(os.Stdout, "ran with args: %v\nUsage: go run diff.go -- [file1] [file2]\n", os.Args)
		os.Exit(1)
	}
	a := os.Args[2]
	b := os.Args[3]
	d, err := diff.CompareGoStructs(a, b)
	if err != nil {
		panic(err)
	}
	if d != "" {
		fmt.Printf("there is structural difference between structs of %s and %s\n%s", a, b, d)
		os.Exit(1)
	}
}
