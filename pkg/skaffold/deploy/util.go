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
	"bufio"
	"bytes"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/sirupsen/logrus"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func parseReleaseInfo(b *bufio.Reader) kubectl.ManifestList {
	r := k8syaml.NewYAMLReader(b)
	var manifests kubectl.ManifestList
	for {
		b, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Infof("error reading line string: %s", err.Error())
			continue
		}
		if i := bytes.Index(b, []byte("apiVersion")); i >= 0 {
			manifests.Append(kubectl.ManifestBytes(b[i:]))
		}
	}
	return manifests
}
