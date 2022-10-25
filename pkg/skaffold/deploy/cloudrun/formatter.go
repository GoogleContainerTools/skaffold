package cloudrun

import (
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
)

type Formatter func(serviceName string) log.Formatter

type cloudRunLogFormatter struct {
	prefix      string
	resources   []string
	outputColor output.Color
}

func newCloudRunFormatter(serviceName string, outputColor output.Color) *cloudRunLogFormatter {
	return &cloudRunLogFormatter{
		prefix:      serviceName,
		outputColor: outputColor,
	}
}
func (formatter *cloudRunLogFormatter) Name() string {
	return formatter.prefix
}
func (formatter *cloudRunLogFormatter) PrintLine(out io.Writer, line string) {
	if output.IsColorable(out) {
		formatter.outputColor.Fprintf(out, "[%s] %s", formatter.prefix, line)
	} else {
		output.Default.Fprintln(out, "[%s] %s", formatter.prefix, line)
	}
}
