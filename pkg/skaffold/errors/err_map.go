/*
Copyright 2020 The Skaffold Authors

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

package errors

import (
	"fmt"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

const (
	// Skaffold commands
	dev   = "dev"
	debug = "debug"
)

var (
	// Deploy errors in deployment phase
	knownDeployProblems = []Problem{
		{
			Regexp:  re("(?i).*unable to connect.*: Get (.*)"),
			ErrCode: proto.StatusCode_DEPLOY_CLUSTER_CONNECTION_ERR,
			Description: func(err error) string {
				matchExp := re("(?i).*unable to connect.*Get (.*)")
				if match := matchExp.FindStringSubmatch(fmt.Sprintf("%s", err)); len(match) >= 2 {
					return fmt.Sprintf("Deploy Failed. Could not connect to cluster %s due to %s", runCtx.KubeContext, match[1])
				}
				return fmt.Sprintf("Deploy Failed. Could not connect to %s cluster.", runCtx.KubeContext)
			},
			Suggestion: suggestDeployFailedAction,
		},
	}
)
