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

package tips

import (
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

// PrintForRun prints tips to the user who has run `skaffold run`.
func PrintForRun(out io.Writer, opts config.SkaffoldOptions) {
	if !opts.Tail {
		printTip(out, "You can also run [skaffold run --tail] to get the logs")
	}
}

// PrintForInit prints tips to the user who has run `skaffold init`.
func PrintForInit(out io.Writer, opts config.SkaffoldOptions) {
	printTip(out, "You can now run [skaffold build] to build the artifacts")
	printTip(out, "or [skaffold run] to build and deploy")
	printTip(out, "or [skaffold dev] to enter development mode, with auto-redeploy")
}

// PrintUseRunVsDeploy prints tips on when to use skaffold run vs deploy.
func PrintUseRunVsDeploy(out io.Writer) {
	printTip(out, "You either need to:")
	printTip(out, "run [skaffold deploy] with [--images TAG] for each pre-built artifact")
	printTip(out, "or [skaffold run] instead, to let Skaffold build, tag and deploy artifacts.")
}

func printTip(out io.Writer, message string) {
	color.Green.Fprintln(out, message)
}
