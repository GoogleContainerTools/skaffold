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

package misc

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// EvaluateEnv evaluates templated environment variables.
func EvaluateEnv(env []string) ([]string, error) {
	var evaluated []string

	for _, kv := range env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) != 2 {
			return nil, fmt.Errorf("invalid env variable: %s, should be using `key=value` form", kv)
		}

		k := kvp[0]
		v := kvp[1]

		value, err := util.ExpandEnvTemplate(v, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to get value for env variable %q: %w", k, err)
		}

		evaluated = append(evaluated, k+"="+value)
	}

	return evaluated, nil
}
