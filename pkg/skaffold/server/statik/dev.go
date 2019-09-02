// +build dev

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

package statik

import (
	"net/http"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
)

var webDir string
var FileSystem = http.Dir(webDir)

func init() {
	color.Green.Fprintln(os.Stdout, "------------------------")
	color.Green.Fprintln(os.Stdout, "       DEV DASH MODE    ")
	color.Green.Fprintln(os.Stdout, "------------------------")
}
