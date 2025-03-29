package commands

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"text/template"

	strs "github.com/buildpacks/pack/internal/strings"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
)

const inspectExtensionTemplate = `
{{ .Location -}}:

Extension:
{{ .Extension }}
`

func inspectAllExtensions(client PackClient, options ...client.InspectExtensionOptions) (string, error) {
	buf := bytes.NewBuffer(nil)
	errArray := []error{}
	for _, option := range options {
		nextResult, err := client.InspectExtension(option)
		if err != nil {
			errArray = append(errArray, err)
			continue
		}

		prefix := determinePrefix(option.ExtensionName, nextResult.Location, option.Daemon)

		output, err := inspectExtensionOutput(nextResult, prefix)
		if err != nil {
			return "", err
		}

		if _, err := buf.Write(output); err != nil {
			return "", err
		}

		if nextResult.Location != buildpack.PackageLocator {
			return buf.String(), nil
		}
	}
	if len(errArray) == len(options) {
		return "", joinErrors(errArray)
	}
	return buf.String(), nil
}

func inspectExtensionOutput(info *client.ExtensionInfo, prefix string) (output []byte, err error) {
	tpl := template.Must(template.New("inspect-extension").Parse(inspectExtensionTemplate))
	exOutput, err := extensionsOutput(info.Extension)
	if err != nil {
		return []byte{}, fmt.Errorf("error writing extension output: %q", err)
	}

	buf := bytes.NewBuffer(nil)
	err = tpl.Execute(buf, &struct {
		Location  string
		Extension string
	}{
		Location:  prefix,
		Extension: exOutput,
	})

	if err != nil {
		return []byte{}, fmt.Errorf("error templating extension output template: %q", err)
	}
	return buf.Bytes(), nil
}

func extensionsOutput(ex dist.ModuleInfo) (string, error) {
	buf := &bytes.Buffer{}

	tabWriter := new(tabwriter.Writer).Init(buf, writerMinWidth, writerPadChar, buildpacksTabWidth, writerPadChar, writerFlags)
	if _, err := fmt.Fprint(tabWriter, "  ID\tNAME\tVERSION\tHOMEPAGE\n"); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(tabWriter, "  %s\t%s\t%s\t%s\n", ex.ID, strs.ValueOrDefault(ex.Name, "-"), ex.Version, strs.ValueOrDefault(ex.Homepage, "-")); err != nil {
		return "", err
	}

	if err := tabWriter.Flush(); err != nil {
		return "", err
	}

	return strings.TrimSuffix(buf.String(), "\n"), nil
}
