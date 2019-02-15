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
	"text/template"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd"
	"github.com/spf13/pflag"
)

const manTemplate = `
### skaffold {{.Name}}

{{.Short}}

` + "```" +
	`
{{.UsageString}}

` + "```\n"

func main() {
	tmpl, err := template.New("test").Parse(manTemplate)
	if err != nil {
		panic(err)
	}

	command := cmd.NewSkaffoldCommand(os.Stdout, os.Stderr)
	for _, command := range command.Commands() {
		err = tmpl.Execute(os.Stdout, command)
		if err != nil {
			panic(err)
		}
		fmt.Println("Env vars:")
		fmt.Println("")
		command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			fmt.Printf("* `%s` (same as --%s)\n", cmd.FlagToEnvVarName(flag), flag.Name)
		})
	}
}
