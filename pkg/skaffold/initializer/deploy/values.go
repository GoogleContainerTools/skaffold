/*
Copyright 2022 The Skaffold Authors

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
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
)

func findValuesFile(chDir string) string {
	vf := filepath.Join(chDir, "values.yaml")
	if _, err := os.Stat(vf); os.IsNotExist(err) {
		chPath := filepath.Join(chDir, "Chart.yaml")
		vf, err = promptValueFile(chPath)
		if err != nil {
			logrus.Warnf("could not find value file for chart at %s. "+
				"This may result in incorrect helm config for this chart", chPath)
			return vf
		}
	}
	return vf
}

func promptValueFile(chPath string) (string, error) {
	chDir := filepath.Dir(chPath)
	vf := valueFile{chDir: chDir}
	qs := []*survey.Question{
		{
			Prompt:    &survey.Input{Message: fmt.Sprintf("Enter values file for chart %s relative to chart dir %s:", chPath, chDir)},
			Validate:  vf.filePathExists,
			Transform: survey.TransformString(vf.path),
		},
	}
	return ask(qs)
}

func ask(qs []*survey.Question) (string, error) {
	var answer string
	if err := survey.Ask(qs, &answer); err != nil {
		return "", err
	}
	return answer, nil
}

type valueFile struct {
	chDir string
}

func (vf valueFile) filePathExists(val interface{}) error {
	if err := survey.Required(val); err != nil {
		return err
	}
	fp := filepath.Join(vf.chDir, fmt.Sprintf("%v", val))
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		// nolint
		return fmt.Errorf("File %s does not exists", fp)
	}
	return nil
}

func (vf valueFile) path(s string) string {
	return filepath.Join(vf.chDir, s)
}
