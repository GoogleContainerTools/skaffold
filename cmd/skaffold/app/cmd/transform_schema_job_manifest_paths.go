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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	olog "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

type jobManifestPathList struct {
	VerifyJobManifestPaths       map[string]string `json:"verifyJobManifestPaths"`
	CustomActionJobManifestPaths map[string]string `json:"customActionJobManifestPaths"`
}

// cmdTransformJobManifestPaths describes the CLI command to diagnose skaffold.
func cmdTransformJobManifestPaths() *cobra.Command {
	return NewCmd("jobManifestPaths").
		WithExample("Get list of jobManifestPaths", "inspect jobManifestPaths list --format json").
		WithExample("Get list of jobManifestPaths targeting a specific configuration", "inspect jobManifestPaths list --profile local --format json").
		WithDescription("Print the list of jobManifestPaths that would be run for a given configuration (default skaffold configuration, specific module, specific profile, etc).").
		WithCommonFlags().
		WithFlags([]*Flag{
			// TODO(aaron-prindle) vvv 2 commands use this, should add to common flags w/ those 2 commands added
			{Value: &outputFile, Name: "output", Shorthand: "o", DefValue: "", Usage: "File to write diagnose result"},
		}).
		WithArgs(func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				olog.Entry(context.TODO()).Errorf("`transform-schema jobManifestPaths` requires exactly one manifest file path argument")
				return errors.New("`transform-schema jobManifestPaths` requires exactly one manifest file path argument")
			}
			return nil
		}, transformJobManifestPaths)
}

func transformJobManifestPaths(ctx context.Context, out io.Writer, args []string) error {
	transformFile := args[0]

	// Open our jsonFile
	jsonFile, err := os.Open(transformFile)
	if err != nil {
		fmt.Println(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result jobManifestPathList
	json.Unmarshal([]byte(byteValue), &result)

	// force absolute path resolution during transform
	opts.MakePathsAbsolute = util.Ptr(true)
	configs, err := getCfgs(ctx, opts)
	if err != nil {
		return err
	}
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	// remove the dependency config references since they have already been imported and will be marshalled together.
	for i := range configs {
		configs[i].(*latest.SkaffoldConfig).Dependencies = nil
		// TRANSFORM HERE
		// ====
		for j := range configs[i].(*latest.SkaffoldConfig).Verify {
			if jobManifestPath, ok := result.VerifyJobManifestPaths[configs[i].(*latest.SkaffoldConfig).Verify[j].Name]; ok {
				// if configs[i].(*latest.SkaffoldConfig).Verify[j].ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath != ""
				configs[i].(*latest.SkaffoldConfig).Verify[j].ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath = jobManifestPath

			}
		}
		// ====
	}

	buf, err := yaml.MarshalWithSeparator(configs)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}
	out.Write(buf)

	return nil
}
