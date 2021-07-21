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

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	timecomp "github.com/GoogleContainerTools/skaffold/hack/time-comparison/metrics-collector"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/sirupsen/logrus"
)

const numBinaries = 2

// TODO(aaron-prindle) figure out these params for users...
const commentText = "// test comment"
const configFile = "config.yaml"

const configFileTemplate = `applications:
- name: {{.AppName}}
  context: ../../examples/{{.AppName}}
  dev:
    command: sh -c "printf "{{.CommentText}}\n" >> {{.FilePath}}"
`

type Config struct {
	Name                  string `yaml:"name,omitempty"`
	DevIterations         int    `yaml:"devIterations,omitempty"`
	FirstSkaffoldFlags    string `yaml:"firstSkaffoldFlags,omitempty"`
	SecondSkaffoldFlags   string `yaml:"secondSkaffoldFlags,omitempty"`
	ExampleAppName        string `yaml:"exampleAppName,omitempty"`
	ExampleFileToEditPath string `yaml:"exampleFileToEditPath,omitempty"`
}

var (
	conf          Config
	yamlInputFile string
)

func init() {
	// flag.StringVar(&configFile, "file", "config.yaml", "path to config file")
	flag.IntVar(&conf.DevIterations, "dev-iterations", 2, "number of dev iterations to run for skaffold.  For one initial loop and one 'inner loop', --dev-iterations=2")
	flag.StringVar(&conf.ExampleAppName, "example-app-name", "microservices", "name of example app under examples/ to use - default is 'microservices'")
	flag.StringVar(&conf.ExampleFileToEditPath, "example-file-to-edit-path", "leeroy-app/app.go", "name of example file to modify during dev session, assumes root ex: workdir=examples/microservices - default is 'leeroy-app/app.go'")
	flag.StringVar(&conf.FirstSkaffoldFlags, "first-skaffold-flags", "", "flag opts to pass to first skaffold binary invocations")
	flag.StringVar(&conf.SecondSkaffoldFlags, "second-skaffold-flags", "", "flag opts to pass to second skaffold binary invocations")
	flag.StringVar(&yamlInputFile, "", "yaml-input-file.yaml", "path to yaml file with input args")

}

func main() {
	ctx := context.Background()
	flag.Parse()

	// if yamlInputFile set, values from that file override flag opts
	if yamlInputFile != "" {
		yamlFile, err := ioutil.ReadFile(yamlInputFile)
		if err != nil {
			logrus.Fatalf("error reading yaml input file: %v ", err)
		}
		err = yaml.Unmarshal(yamlFile, conf)
		if err != nil {
			logrus.Fatalf("error unmarshalling yaml input file: %v", err)
		}
		logrus.Infof("unmarshalled yaml input file into Config struct: %+v", conf)
	}

	if len(flag.Args()) < numBinaries+1 {
		// time-comparison --first-skaffold-flags="--build-concurrency=true" \
		// --second-skaffold-flags="--build-concurrency=false" \
		// skaffold-1 skaffold-2 microservices out.txt
		logrus.Fatalf("time-comparison expects input of the form: timer-comparison /usr/bin/bin1 /usr/bin/bin2 output.txt")
	}

	if err := validateArgs(flag.Args()); err != nil {
		logrus.Fatal(err)
	}
	commentPath := flag.Args()[len(flag.Args())-1]
	skaffoldFlags := []string{conf.FirstSkaffoldFlags, conf.SecondSkaffoldFlags}

	var b bytes.Buffer
	workDir, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}

	configFileInputs := struct {
		AppName     string
		CommentText string
		FilePath    string
	}{
		conf.ExampleAppName,
		commentText,
		conf.ExampleFileToEditPath,
	}

	t, err := template.New(configFile).Parse(configFileTemplate)
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	var w bytes.Buffer
	err = t.Execute(&w, configFileInputs)
	if err != nil {
		log.Print("template executing error: ", err) //log it
	}

	// write file
	ioutil.WriteFile(configFile, w.Bytes(), 0644)
	defer func() {
		if err := os.Remove(configFile); err != nil {
			logrus.Fatal(err)
		}
	}()

	for i := 0; i < numBinaries; i++ {
		eventsFileAbsPath := filepath.Join(workDir, "events-"+strconv.Itoa(i))
		tci := timecomp.TimeComparisonInput{
			ConfigFile:         configFile,
			SkaffoldBinaryPath: flag.Args()[i],
			EventsFileAbsPath:  eventsFileAbsPath,
			SkaffoldFlags:      skaffoldFlags[i],
			// cleanup             bool
		}
		if err := timecomp.CollectTimingInformation(tci); err != nil {
			logrus.Fatal(err)
		}
		mtrcs, err := SkaffoldRunMetrics(ctx, eventsFileAbsPath)
		if err != nil {
			logrus.Fatal(err)
		}

		binFile, err := os.Stat(flag.Args()[i])
		if err != nil {
			logrus.Fatal(err)
		}

		//  mtrcs[1] is the "inner" dev loop metrics & mtrcs[0] is the initial dev loop
		tco := TimeComparisonOutput{
			binaryPath: flag.Args()[i],
			binarySize: binFile.Size(),
			// loopMetrics: mtrcs,
			innerLoopBuildTime:       mtrcs[1].buildTime,
			innerLoopDeployTime:      mtrcs[1].deployTime,
			innerLoopStatusCheckTime: mtrcs[1].statusCheckTime,
			innerLoopTotalTime:       mtrcs[1].buildTime + mtrcs[1].deployTime + mtrcs[1].statusCheckTime,
		}
		fmt.Fprint(&b, tco.String())
	}
	logrus.Infof("writing time comparison information to path %v with text:\n%v ", commentPath, b.String())
	if err := ioutil.WriteFile(filepath.Join(workDir, commentPath), b.Bytes(), 0644); err != nil {
		logrus.Fatal(err)
	}
}
