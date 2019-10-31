// +build release

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

package cmd

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/rakyll/statik/fs"

	//required for rakyll/statik embedded content
	_ "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/credits/statik"
)

func exportCredits(out io.Writer) error {
	statikFS, err := fs.New()
	if err != nil {
		log.Fatalf("error opening embedded filesystem: %s", err)
		return err
	}
	err = fs.Walk(statikFS, "/", func(filePath string, fileInfo os.FileInfo, err error) error {
		newPath := path.Join(creditsPath, filePath)
		if fileInfo.IsDir() {
			err := os.Mkdir(newPath, 0755)
			if err != nil && !os.IsExist(err) {
				log.Fatalf("error creating directory %s: %s", newPath, err)
				return err
			}
		}
		if !fileInfo.IsDir() {
			file, err := statikFS.Open(filePath)
			if err != nil {
				log.Fatalf("error opening %s in embedded filesystem: %s", filePath, err)
				return err
			}
			buf, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("error reading %s in embedded filesystem: %s", filePath, err)
				return err
			}
			err = ioutil.WriteFile(newPath, buf, 0664)
			if err != nil {
				log.Fatalf("error writing %s to %s: %s", filePath, newPath, err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
		return err
	}
	s, err := filepath.Abs(creditsPath)
	if err != nil {
		log.Printf("Successfully exported third party notices to %s", creditsPath)
	}
	log.Printf("Successfully exported third party notices to %s", s)
	return nil
}
