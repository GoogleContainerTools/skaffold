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

package validate

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const NumBinaries = 2

func Args(args []string) error {
	if len(args) < NumBinaries+2 {
		logrus.Fatalf("comparisonstats expects input of the form: $ comparisonstats /usr/bin/bin1 /usr/bin/bin2 helm-deployment main.go \"//per-dev-iteration-comment\"")
	}

	if err := validateBinaries(args[1:NumBinaries]); err != nil {
		return err
	}

	if err := validateExampleAppNameAndSrcFile(args[NumBinaries], args[1+NumBinaries]); err != nil {
		return err
	}

	return nil
}

func validateBinaries(binpaths []string) error {
	for _, binpath := range binpaths {
		_, err := os.Stat(binpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateExampleAppNameAndSrcFile(exampleAppName, exampleSrcFile string) error {
	fp := filepath.Join("examples/", exampleAppName)
	if filepath.IsAbs(exampleAppName) {
		fp = exampleAppName
	}
	_, err := os.Stat(fp)
	if err != nil {
		return err
	}

	fp = filepath.Join("examples/", exampleAppName, exampleSrcFile)
	if filepath.IsAbs(exampleAppName) {
		fp = exampleAppName
	}
	_, err = os.Stat(fp)
	if err != nil {
		return err
	}
	return nil
}
