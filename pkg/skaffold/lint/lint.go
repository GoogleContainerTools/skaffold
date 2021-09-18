/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliecf.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lint

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
)

func Lint(ctx context.Context, out io.Writer, opts inspect.Options) error {
	skaffoldYamlRuleList, err := GetSkaffoldYamlsList(ctx, out, opts)
	if err != nil {
		return err
	}
	dockerfileRuleList, err := GetDockerfilesList(ctx, out, opts)
	if err != nil {
		return err
	}
	k8sYamlRuleList, err := GetK8sYamlsList(ctx, out, opts)
	if err != nil {
		return err
	}
	recList := RuleList{
		LinterResultList: []Result{},
	}
	recList.LinterResultList = append(recList.LinterResultList, skaffoldYamlRuleList.SkaffoldYamlRules...)
	recList.LinterResultList = append(recList.LinterResultList, dockerfileRuleList.DockerfileRules...)
	recList.LinterResultList = append(recList.LinterResultList, k8sYamlRuleList.K8sYamlRules...)

	//   output flattened list
	formatter := OutputFormatter(out, opts.OutFormat)
	return formatter.Write(recList)
}
