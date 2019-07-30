/*
Copyright 2018 The Knative Authors

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

package configmap

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// Load reads the "Data" of a ConfigMap from a particular VolumeMount.
func Load(p string) (map[string]string, error) {
	data := make(map[string]string)
	err := filepath.Walk(p, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		for info.Mode()&os.ModeSymlink != 0 {
			dirname := filepath.Dir(p)
			p, err = os.Readlink(p)
			if err != nil {
				return err
			}
			if !filepath.IsAbs(p) {
				p = path.Join(dirname, p)
			}
			info, err = os.Lstat(p)
			if err != nil {
				return err
			}
		}
		if info.IsDir() {
			return nil
		}
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		data[info.Name()] = string(b)
		return nil
	})
	return data, err
}
