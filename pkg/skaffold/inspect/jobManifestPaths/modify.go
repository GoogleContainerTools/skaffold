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

package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

var (
	getCfgs = parser.GetAllConfigs
)

func Modify(ctx context.Context, out io.Writer, opts config.SkaffoldOptions, inputFile, outputFile string) error {
	// Open our jsonFile
	jsonFile, err := os.Open(inputFile)
	if err != nil {
		log.Entry(context.TODO()).Error(err)
		return err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var result jobManifestPathList
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		log.Entry(context.TODO()).Error(err)
		return err
	}

	// force absolute path resolution during modify
	opts.MakePathsAbsolute = util.Ptr(true)
	configs, err := getCfgs(ctx, opts)
	if err != nil {
		log.Entry(context.TODO()).Error(err)
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
		for j := range configs[i].(*latest.SkaffoldConfig).Verify {
			if jobManifestPath, ok := result.VerifyJobManifestPaths[configs[i].(*latest.SkaffoldConfig).Verify[j].Name]; ok {
				configs[i].(*latest.SkaffoldConfig).Verify[j].ExecutionMode.KubernetesClusterExecutionMode.JobManifestPath = jobManifestPath
			}
		}

		for j := range configs[i].(*latest.SkaffoldConfig).CustomActions {
			if jobManifestPath, found := result.CustomActionJobManifestPaths[configs[i].(*latest.SkaffoldConfig).CustomActions[j].Name]; found {
				if configs[i].(*latest.SkaffoldConfig).CustomActions[j].ExecutionModeConfig.KubernetesClusterExecutionMode == nil {
					configs[i].(*latest.SkaffoldConfig).CustomActions[j].ExecutionModeConfig.KubernetesClusterExecutionMode = &latest.KubernetesClusterVerifier{}
				}

				configs[i].(*latest.SkaffoldConfig).CustomActions[j].ExecutionModeConfig.KubernetesClusterExecutionMode.JobManifestPath = jobManifestPath
			}
		}
	}

	buf, err := yaml.MarshalWithSeparator(configs)
	if err != nil {
		return fmt.Errorf("marshalling configuration: %w", err)
	}
	out.Write(buf)

	return nil
}
