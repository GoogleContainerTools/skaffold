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

package lint

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var realWorkDir = util.RealWorkDir

func GetAllLintResults(ctx context.Context, opts Options, dockerCfg docker.Config) ([]Result, error) {
	results := []Result{}
	skaffoldYamlResults, err := GetSkaffoldYamlsLintResults(ctx, opts)
	if err != nil {
		return nil, err
	}
	results = append(results, *skaffoldYamlResults...)

	if dockerCfg != nil {
		dockerfileCommandResults, err := GetDockerfilesLintResults(ctx, opts, dockerCfg)
		if err != nil {
			return nil, err
		}
		results = append(results, *dockerfileCommandResults...)
	}

	k8sManifestResults, err := GetK8sManifestsLintResults(ctx, opts)
	if err != nil {
		return nil, err
	}
	results = append(results, *k8sManifestResults...)
	return results, nil
}

func Lint(ctx context.Context, out io.Writer, opts Options, dockerCfg docker.Config) error {
	results, err := GetAllLintResults(ctx, opts, dockerCfg)
	if err != nil {
		return err
	}
	// output flattened list
	if opts.OutFormat == JSONOutput {
		// need to remove some fields that cannot be serialized in the Rules of the Results
		for _, res := range results {
			res.Rule.ExplanationPopulator = nil
			res.Rule.LintConditions = nil
		}
	}
	formatter := OutputFormatter(out, opts.OutFormat)
	return formatter.Write(results)
}
