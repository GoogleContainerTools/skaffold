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

package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

type Copy struct {
	Namespace string
	PodName   string
	Container string
	SrcPath   string
	DestPath  string
}

func (c Copy) CopyAndExtractTarGzInPod(client kubernetes.Interface) error {
	if filepath.Ext(c.SrcPath) != ".gz" {
		return errors.New("SrcPath must end with .gz")
	}

	f, err := os.Open(c.SrcPath)
	if err != nil {
		return errors.Wrap(err, "opening tar file")
	}
	defer f.Close()

	tarCmd := []string{"tar", "-zxf", "-", "-C", c.DestPath}

	exec := Exec{
		Namespace: c.Namespace,
		PodName:   c.PodName,
		Container: c.Container,

		Command: tarCmd,
		Stdin:   f,
	}

	return exec.Exec(client)
}
