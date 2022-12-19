/*
Copyright 2022 The Skaffold Authors

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

package cloudrun

import (
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
)

type LogFormatter struct {
	prefix      string
	outputColor output.Color
}

func (formatter *LogFormatter) Name() string {
	return formatter.prefix
}
func (formatter *LogFormatter) PrintLine(out io.Writer, line string) {
	if output.IsColorable(out) {
		formatter.outputColor.Fprintf(out, "[%s] %s", formatter.prefix, line)
	} else {
		output.Default.Fprintf(out, "[%s] %s", formatter.prefix, line)
	}
}
