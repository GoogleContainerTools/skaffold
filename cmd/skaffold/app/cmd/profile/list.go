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

package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/parser"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"gopkg.in/yaml.v3"
)

func List(ctx context.Context, out io.Writer) error {
	return list(ctx, out, filename, outputType)
}

func list(ctx context.Context, out io.Writer, configurationFile, outputType string) error {
	cfgs, err := parser.GetConfigSet(
		ctx, config.SkaffoldOptions{
			ConfigurationFile: configurationFile,
		},
	)

	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	profiles := make([]*latest.Profile, 0)
	for _, cfg := range cfgs {
		for _, profile := range cfg.Profiles {
			profiles = append(profiles, &profile)
		}
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no profiles found in %s", configurationFile)
	}

	switch outputType {
	case "yaml":
		return printYAML(out, profiles)
	case "json":
		return printJSON(out, profiles)
	case "plain":
		return printPlain(out, profiles)
	default:
		return fmt.Errorf(`invalid output type: %q. Must be "plain" or "json"`, outputType)
	}
}

func printYAML(out io.Writer, profiles []*latest.Profile) error {
	return yaml.NewEncoder(out).Encode(profiles)
}

func printJSON(out io.Writer, profiles []*latest.Profile) error {
	return json.NewEncoder(out).Encode(profiles)
}

func printPlain(out io.Writer, profiles []*latest.Profile) error {
	lines := make([]string, len(profiles))
	for i, profile := range profiles {
		activations := activationsToString(profile.Activation)

		lines[i] = fmt.Sprintf(
			"- %s\n    Activation: %+v\n    RequiresAllActivations: %v\n",
			profile.Name,
			activations,
			profile.RequiresAllActivations,
		)
	}
	_, err := out.Write([]byte(strings.Join(lines, "\n")))

	return err
}

func activationsToString(activations []latest.Activation) string {
	res := make([]string, len(activations))
	activationStringBuilder := strings.Builder{}

	for i, activation := range activations {
		activationStringBuilder.Reset()
		if activation.KubeContext != "" {
			activationStringBuilder.WriteString(fmt.Sprintf("kubeContext:%s ", activation.KubeContext))
		}
		if activation.Command != "" {
			activationStringBuilder.WriteString(fmt.Sprintf("command:%s ", activation.Command))
		}
		if activation.Env != "" {
			activationStringBuilder.WriteString(fmt.Sprintf("env:%s ", activation.Env))
		}

		res[i] = strings.TrimSpace(activationStringBuilder.String())
	}

	return fmt.Sprintf("%v", res)
}
