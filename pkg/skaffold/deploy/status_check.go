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

package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/pkg/errors"
)

func StatusCheck(ctx context.Context, out io.Writer, runCtx runcontext.RunContext) error {
	kubeCtl := kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}
	deployments, err := getDeployments(ctx, kubeCtl)
	fmt.Println(deployments)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	return nil
}

func getDeployments(ctx context.Context, k kubectl.CLI) ([]string, error) {
	b, err := k.RunOut(ctx, nil, "get", []string{"deployments"}, "--output", "jsonpath='{.items[*].metadata.name}'")
	if err != nil {
		return nil, err
	}
	s := []string{}
	if len(string(b)) > 0 {
		s = strings.Split(string(b), " ")
	}
	return s, nil

}
