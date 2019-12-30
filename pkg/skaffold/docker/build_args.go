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

package docker

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// GetBuildArgs gives the build args flags for docker build.
func GetBuildArgs(a *latest.DockerArtifact) ([]string, error) {
	var args []string

	buildArgs, err := EvaluateBuildArgs(a.BuildArgs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to evaluate build args")
	}

	var keys []string
	for k := range buildArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		args = append(args, "--build-arg")

		v := buildArgs[k]
		if v == nil {
			args = append(args, k)
		} else {
			args = append(args, fmt.Sprintf("%s=%s", k, *v))
		}
	}

	for _, from := range a.CacheFrom {
		args = append(args, "--cache-from", from)
	}

	if a.Target != "" {
		args = append(args, "--target", a.Target)
	}

	if a.NetworkMode != "" {
		args = append(args, "--network", strings.ToLower(a.NetworkMode))
	}

	if a.NoCache {
		args = append(args, "--no-cache")
	}

	return args, nil
}

// EvaluateBuildArgs evaluates templated build args.
func EvaluateBuildArgs(args map[string]*string) (map[string]*string, error) {
	if args == nil {
		return nil, nil
	}

	evaluated := map[string]*string{}
	for k, v := range args {
		if v == nil {
			evaluated[k] = nil
			continue
		}

		tmpl, err := util.ParseEnvTemplate(*v)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parse template for build arg: %s=%s", k, *v)
		}

		value, err := util.ExecuteEnvTemplate(tmpl, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get value for build arg: %s", k)
		}
		evaluated[k] = &value
	}

	return evaluated, nil
}
