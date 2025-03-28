package commands

import (
	"bytes"
	"html/template"
	"sort"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/pkg/logging"
)

type suggestedStack struct {
	ID          string
	Description string
	Maintainer  string
	BuildImage  string
	RunImage    string
}

var suggestedStacks = []suggestedStack{
	{
		ID:          "Deprecation Notice",
		Description: "Stacks are deprecated in favor of using BuildImages and RunImages directly, but will continue to be supported throughout all of 2023 and 2024 if not longer. Please see our docs for more details- https://buildpacks.io/docs/concepts/components/stack",
		Maintainer:  "CNB",
	},
	{
		ID:          "heroku-20",
		Description: "The official Heroku stack based on Ubuntu 20.04",
		Maintainer:  "Heroku",
		BuildImage:  "heroku/heroku:20-cnb-build",
		RunImage:    "heroku/heroku:20-cnb",
	},
	{
		ID:          "io.buildpacks.stacks.jammy",
		Description: "A minimal Paketo stack based on Ubuntu 22.04",
		Maintainer:  "Paketo Project",
		BuildImage:  "paketobuildpacks/build-jammy-base",
		RunImage:    "paketobuildpacks/run-jammy-base",
	},
	{
		ID:          "io.buildpacks.stacks.jammy",
		Description: "A large Paketo stack based on Ubuntu 22.04",
		Maintainer:  "Paketo Project",
		BuildImage:  "paketobuildpacks/build-jammy-full",
		RunImage:    "paketobuildpacks/run-jammy-full",
	},
	{
		ID:          "io.buildpacks.stacks.jammy.tiny",
		Description: "A tiny Paketo stack based on Ubuntu 22.04, similar to distroless",
		Maintainer:  "Paketo Project",
		BuildImage:  "paketobuildpacks/build-jammy-tiny",
		RunImage:    "paketobuildpacks/run-jammy-tiny",
	},
	{
		ID:          "io.buildpacks.stacks.jammy.static",
		Description: "A static Paketo stack based on Ubuntu 22.04, similar to distroless",
		Maintainer:  "Paketo Project",
		BuildImage:  "paketobuildpacks/build-jammy-static",
		RunImage:    "paketobuildpacks/run-jammy-static",
	},
}

func stackSuggest(logger logging.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "suggest",
		Args:    cobra.NoArgs,
		Short:   "(deprecated) List the recommended stacks",
		Example: "pack stack suggest",
		RunE: logError(logger, func(*cobra.Command, []string) error {
			Suggest(logger)
			return nil
		}),
	}

	return cmd
}

func Suggest(log logging.Logger) {
	sort.Slice(suggestedStacks, func(i, j int) bool { return suggestedStacks[i].ID < suggestedStacks[j].ID })
	tmpl := template.Must(template.New("").Parse(`Stacks maintained by the community:
{{- range . }}

    Stack ID: {{ .ID }}
    Description: {{ .Description }}
    Maintainer: {{ .Maintainer }}
    Build Image: {{ .BuildImage }}
    Run Image: {{ .RunImage }}
{{- end }}
`))

	buf := &bytes.Buffer{}
	tmpl.Execute(buf, suggestedStacks)
	log.Info(buf.String())
}
