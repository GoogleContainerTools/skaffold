package main

import (
	"fmt"
	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd"
	"github.com/spf13/pflag"
	"os"
	"text/template"
)

const manTemplate = `### skaffold {{.Name}}

{{.Short}}

`+ "```" +
`
{{.UsageString}}

`+ "```\n"


func main() {
	command := cmd.NewSkaffoldCommand(os.Stdout, os.Stderr)
	for _, command :=  range command.Commands() {
		tmpl, err := template.New("test").Parse(manTemplate)
		if err != nil { panic(err) }
		err = tmpl.Execute(os.Stdout, command)
		if err != nil { panic(err) }
		fmt.Println("Env vars:")
		fmt.Println("")
		command.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			fmt.Printf("* `%s` (same as --%s)\n", cmd.FlagToEnvVarName(flag), flag.Name)
		})
	}
}
