/*
Copyright 2021 The Skaffold Authors

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

package log

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type Formatter interface {
	Name() string

	PrintLine(io.Writer, string)
}

func ParseJSON(config latest.JSONParseConfig, line string) string {
	if len(config.Fields) == 0 {
		return line
	}

	js := map[string]interface{}{}
	trimmed := strings.Trim(line, "\n")
	if err := json.Unmarshal([]byte(trimmed), &js); err != nil {
		olog.Entry(context.TODO()).Debugf("failed to unmarshal json: %s", err)
		return line
	}

	result := ""
	for _, field := range config.Fields {
		if val, ok := js[field]; ok {
			result += fmt.Sprintf("%s: %v, ", field, val)
		}
	}

	// If none of the fields specified were in the json object, just return the original line
	if result == "" {
		return line
	}
	return strings.TrimSuffix(result, ", ") + "\n"
}
