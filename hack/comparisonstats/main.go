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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/devrunner"
	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/events"
	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/types"
	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/validate"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// comparisonstats usage example:
// $ comparisonstats --first-skaffold-flags="--build-concurrency=0" \
//   --second-skaffold-flags="--build-concurrency=1" \
//   /path/skaffold-1 /path/skaffold-2 helm-deployment main.go "//per-dev-iteration-comment"

var (
	warmupRuns        int
	summaryOutputPath string
	yamlInputFile     string
	conf              = &types.Config{}
)

func init() {
	flag.Int64Var(&conf.DevIterations, "dev-iterations", 2, "number of dev iterations to run for skaffold.  For one initial loop and one 'inner loop', --dev-iterations=2")
	flag.StringVar(&summaryOutputPath, "summary-output-path", "", "path to file to write summary output to")
	flag.StringVar(&conf.CommentText, "comment-text", "//per-dev-iteration-comment", "text to append to the specified 'ExampleSrcFile' during each skaffold dev loop")
	flag.StringVar(&conf.FirstSkaffoldFlags, "first-skaffold-flags", "", "flag opts to pass to first skaffold binary invocations")
	flag.StringVar(&conf.SecondSkaffoldFlags, "second-skaffold-flags", "", "flag opts to pass to second skaffold binary invocations")
	flag.StringVar(&yamlInputFile, "yaml-input-file", "", "path to yaml file with input args")
	flag.IntVar(&warmupRuns, "warmup-runs", 0, "Number of warmup runs to perform (defaults to 0)")
}

func main() {
	ctx := context.Background()
	flag.Parse()

	if err := validate.ValidateArgs(flag.Args()); err != nil {
		logrus.Fatal(err)
	}
	cmdArgs := types.ParseComparisonStatsCmdArgs(flag.Args())
	skaffoldFlags := []string{conf.FirstSkaffoldFlags, conf.SecondSkaffoldFlags}
	for i := 0; i < warmupRuns; i++ {
		cmdArgs.SkaffoldBinaries = append([]string{cmdArgs.SkaffoldBinaries[0]}, cmdArgs.SkaffoldBinaries...)
		skaffoldFlags = append([]string{skaffoldFlags[0]}, skaffoldFlags...)
	}
	conf.ExampleAppName = cmdArgs.ExampleAppName
	conf.ExampleSrcFile = cmdArgs.ExampleSrcFile

	// if yamlInputFile set, update values from yaml file to override flag opts
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

	var b bytes.Buffer
	for i := 0; i < len(cmdArgs.SkaffoldBinaries); i++ {
		uid, _ := uuid.NewUUID()
		random := uid.String()
		eventsFileAbsPath := filepath.Join(os.TempDir(), fmt.Sprintf("events-%d-%s", i, random))
		skaffoldBinaryPath := cmdArgs.SkaffoldBinaries[i]
		devIterations := conf.DevIterations
		if i < warmupRuns {
			devIterations = 2
		}
		app := types.Application{
			Name:          conf.ExampleAppName,
			Context:       fmt.Sprintf("examples/%s", conf.ExampleAppName),
			Dev:           types.Dev{Command: fmt.Sprintf("printf \"%s\\n\" >> %s", conf.CommentText, conf.ExampleSrcFile)},
			DevIterations: devIterations,
		}
		devInfo, err := devrunner.Dev(ctx, app, skaffoldBinaryPath, eventsFileAbsPath, skaffoldFlags[i])
		if err != nil {
			logrus.Fatal(err)
		}
		defer events.Cleanup(eventsFileAbsPath)

		if i < warmupRuns {
			continue

		}
		eventDurations, err := events.ParseEventDuration(ctx, eventsFileAbsPath)
		if err != nil {
			logrus.Fatal(err)
		}

		binFile, err := os.Stat(skaffoldBinaryPath)
		if err != nil {
			logrus.Fatal(err)
		}

		ra := types.ComparisonStatsSummary{
			CmdArgs:               devInfo.CmdArgs,
			BinaryPath:            skaffoldBinaryPath,
			BinarySize:            binFile.Size(),
			DevIterations:         conf.DevIterations,
			DevLoopEventDurations: eventDurations,
		}
		fmt.Fprint(&b, ra.String())
	}

	logrus.Infof("comparison summary information:\n%v ", b.String())

	workDir, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}
	if summaryOutputPath != "" {
		logrus.Infof("writing summary information to path %v", summaryOutputPath)
		if err := ioutil.WriteFile(filepath.Join(workDir, summaryOutputPath), b.Bytes(), 0644); err != nil {
			logrus.Fatal(err)
		}
	}

}
