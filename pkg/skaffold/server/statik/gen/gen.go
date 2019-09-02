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
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

//this file basically does this:
// test `which statik` || GOFLAGS='' go get github.com/rakyll/statik
func main() {
	_, err := exec.LookPath("statik")

	if err != nil {
		switch err := err.(type) {
		case *exec.Error:
			if exec.ErrNotFound == err.Err {
				fmt.Println("Couldn't find statik, downloading...")
				os.Setenv("GOFLAGS", "")
				cmd := exec.Command("go", "get", "github.com/rakyll/statik")
				fmt.Printf("Running %s\n", cmd.Args)
				out, err := util.RunCmdOut(cmd)
				if err != nil {
					panic(fmt.Sprintf("error getting statik: %s, %s", err, out))
				}
			} else {
				panic(fmt.Sprintf("can't find statik: %s vs %s", err.Err, os.ErrNotExist))
			}
		default:
			panic(fmt.Sprintf("can't find statik: %s", err))
		}
	}

	cmd := exec.Command("statik", os.Args...)
	fmt.Printf("Running %v\n", cmd.Args)
	err = cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("error running statik: %s", err))
	}
}
