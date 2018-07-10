/*
Copyright 2018 The Skaffold Authors

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

package knative

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const buildCRDNamespace = "knative-build"

// testBuildCRD verifies that the Build CRD is installed.
func testBuildCRD() error {
	client, err := kubernetes.GetClientset()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	_, err = client.CoreV1().Namespaces().Get(buildCRDNamespace, metav1.GetOptions{})
	return err
}
