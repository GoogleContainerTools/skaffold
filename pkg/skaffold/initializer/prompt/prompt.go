/*
Copyright 2020 The Skaffold Authors

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

package prompt

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	BuildConfigFunc         = buildConfig
	PortForwardResourceFunc = portForwardResource
	askOne                  = survey.AskOne
	ask                     = survey.Ask
)

func buildConfig(image string, choices []string) (string, error) {
	var selectedBuildConfig string
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Choose the builder to build image %s", image),
		Options:  choices,
		PageSize: 15,
	}
	err := survey.AskOne(prompt, &selectedBuildConfig, nil)
	if err != nil {
		return "", err
	}

	return selectedBuildConfig, nil
}

func WriteSkaffoldConfig(out io.Writer, pipeline []byte, generatedManifests map[string][]byte, filePath string) (bool, error) {
	fmt.Fprintln(out, string(pipeline))

	for path, manifest := range generatedManifests {
		fmt.Fprintln(out, path, "-", string(manifest))
	}

	manifestString := ""
	if len(generatedManifests) > 0 {
		manifestString = ", along with the generated k8s manifests,"
	}

	var response bool
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Do you want to write this configuration%s to %s?", manifestString, filePath),
	}
	err := askOne(prompt, &response, nil)
	if err != nil {
		return true, fmt.Errorf("reading user confirmation: %w", err)
	}

	return !response, nil
}

// PortForwardResource prompts the user to give a port to forward the current resource on
func portForwardResource(out io.Writer, imageName string) (int, error) {
	var response string
	prompt := &survey.Question{
		Prompt: &survey.Input{Message: fmt.Sprintf("Select port to forward for %s (leave blank for none): ", imageName)},
		Validate: func(val interface{}) error {
			str := val.(string)
			if _, err := strconv.Atoi(str); err != nil && str != "" {
				return errors.New("response must be a number, or empty")
			}
			return nil
		},
	}
	err := ask([]*survey.Question{prompt}, &response)
	if err != nil {
		return 0, fmt.Errorf("reading user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" {
		return 0, nil
	}

	responseInt, _ := strconv.Atoi(response)
	return responseInt, nil
}

// ConfirmInitOptions prompts the user to confirm that they are okay with what skaffold will do if they
// run with the current config
func ConfirmInitOptions(out io.Writer, config *latest.SkaffoldConfig) (bool, error) {
	builders := strings.Join(util.ListBuilders(&config.Build), ",")
	deployers := strings.Join(util.ListDeployers(&config.Deploy), ",")

	fmt.Fprintf(out, `If you choose to continue, skaffold will do the following:
  - Create a skaffold config file for you
  - Build your application using %s
  - Deploy your application to your current kubernetes context using %s

`, builders, deployers)

	var response bool
	prompt := &survey.Confirm{
		Message: "Would you like to continue?",
	}
	err := askOne(prompt, &response, nil)
	if err != nil {
		return true, err
	}

	// invert response because "no" == done and "yes" == !done
	return !response, nil
}
