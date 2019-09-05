// +build gofuzz

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

package fuzz

import (
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	_ "github.com/dvyukov/go-fuzz/go-fuzz-dep"
)

// FuzzParseConfig tests configuration file parsing.
func FuzzParseConfig(fuzz []byte) int {
	file, err := ioutil.TempFile("", "fuzzconfig")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	_, err = file.Write(fuzz)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	_, err = config.ReadConfigFileNoCache(file.Name())
	if err != nil {
		return 0
	}
	return 1
}
