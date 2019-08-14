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

package util

import (
	"strings"

	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/pkg/errors"
)


// GetDeployNamespace gets the namespace the current skaffold command will deploy resources into
// Resolution order is:
// + The namespace passed on the command line
// + Current kube context's namespace
// + default if none is present.
func GetDeployNamespace(cmdNamespace string) (string, error) {
	if ns := strings.TrimSpace(cmdNamespace); ns != "" {
		return ns, nil
	}
	// Get current kube context's namespace
		config, err := kubectx.CurrentConfig()
		if err != nil {
			return "", errors.Wrap(err, "getting k8s configuration")
		}

		current, ok := config.Contexts[config.CurrentContext]
		if ok && strings.TrimSpace(current.Namespace) != "" {
			return strings.TrimSpace(current.Namespace), nil
		}
		return "default", nil
}

