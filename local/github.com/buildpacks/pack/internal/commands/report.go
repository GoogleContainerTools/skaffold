package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/logging"
)

func Report(logger logging.Logger, version, cfgPath string) *cobra.Command {
	var explicit bool

	cmd := &cobra.Command{
		Use:     "report",
		Args:    cobra.NoArgs,
		Short:   "Display useful information for reporting an issue",
		Example: "pack report",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			var buf bytes.Buffer
			err := generateOutput(&buf, version, cfgPath, explicit)
			if err != nil {
				return err
			}

			logger.Info(buf.String())

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&explicit, "explicit", "e", false, "Print config without redacting information")
	AddHelpFlag(cmd, "report")
	return cmd
}

func generateOutput(writer io.Writer, version, cfgPath string, explicit bool) error {
	tpl := template.Must(template.New("").Parse(`Pack:
  Version:  {{ .Version }}
  OS/Arch:  {{ .OS }}/{{ .Arch }}

Default Lifecycle Version:  {{ .DefaultLifecycleVersion }}

Supported Platform APIs:  {{ .SupportedPlatformAPIs }}

Config:
{{ .Config -}}`))

	configData := ""
	if data, err := os.ReadFile(filepath.Clean(cfgPath)); err != nil {
		configData = fmt.Sprintf("(no config file found at %s)", cfgPath)
	} else {
		var padded strings.Builder

		for _, line := range strings.Split(string(data), "\n") {
			if !explicit {
				line = sanitize(line)
			}
			_, _ = fmt.Fprintf(&padded, "  %s\n", line)
		}
		configData = strings.TrimRight(padded.String(), " \n")
	}

	platformAPIs := strings.Join(build.SupportedPlatformAPIVersions.AsStrings(), ", ")

	return tpl.Execute(writer, map[string]string{
		"Version":                 version,
		"OS":                      runtime.GOOS,
		"Arch":                    runtime.GOARCH,
		"DefaultLifecycleVersion": builder.DefaultLifecycleVersion,
		"SupportedPlatformAPIs":   platformAPIs,
		"Config":                  configData,
	})
}

func sanitize(line string) string {
	re := regexp.MustCompile(`"(.*?)"`)
	redactedString := `"[REDACTED]"`
	sensitiveFields := []string{
		"default-builder-image",
		"image",
		"mirrors",
		"name",
		"url",
	}
	for _, field := range sensitiveFields {
		if strings.HasPrefix(strings.TrimSpace(line), field) {
			return re.ReplaceAllString(line, redactedString)
		}
	}

	return line
}
