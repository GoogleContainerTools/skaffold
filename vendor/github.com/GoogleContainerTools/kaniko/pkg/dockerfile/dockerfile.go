/*
Copyright 2018 Google LLC

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

package dockerfile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// Stages reads the Dockerfile, validates it's contents, and returns stages
func Stages(dockerfilePath, target string) ([]instructions.Stage, error) {
	d, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil, err
	}

	stages, err := Parse(d)
	if err != nil {
		return nil, err
	}
	if err := ValidateTarget(stages, target); err != nil {
		return nil, err
	}
	ResolveStages(stages)
	return stages, nil
}

// Parse parses the contents of a Dockerfile and returns a list of commands
func Parse(b []byte) ([]instructions.Stage, error) {
	p, err := parser.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	stages, _, err := instructions.Parse(p.AST)
	if err != nil {
		return nil, err
	}
	return stages, err
}

func ValidateTarget(stages []instructions.Stage, target string) error {
	if target == "" {
		return nil
	}
	for _, stage := range stages {
		if stage.Name == target {
			return nil
		}
	}
	return fmt.Errorf("%s is not a valid target build stage", target)
}

// ResolveStages resolves any calls to previous stages with names to indices
// Ex. --from=second_stage should be --from=1 for easier processing later on
func ResolveStages(stages []instructions.Stage) {
	nameToIndex := make(map[string]string)
	for i, stage := range stages {
		index := strconv.Itoa(i)
		if stage.Name != index {
			nameToIndex[stage.Name] = index
		}
		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.CopyCommand:
				if c.From != "" {
					if val, ok := nameToIndex[c.From]; ok {
						c.From = val
					}
				}
			}
		}
	}
}

// ParseCommands parses an array of commands into an array of instructions.Command; used for onbuild
func ParseCommands(cmdArray []string) ([]instructions.Command, error) {
	var cmds []instructions.Command
	cmdString := strings.Join(cmdArray, "\n")
	ast, err := parser.Parse(strings.NewReader(cmdString))
	if err != nil {
		return nil, err
	}
	for _, child := range ast.AST.Children {
		cmd, err := instructions.ParseCommand(child)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

// SaveStage returns true if the current stage will be needed later in the Dockerfile
func SaveStage(index int, stages []instructions.Stage) bool {
	for stageIndex, stage := range stages {
		if stageIndex <= index {
			continue
		}
		if stage.Name == stages[index].BaseName {
			return true
		}
		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.CopyCommand:
				if c.From == strconv.Itoa(index) {
					return true
				}
			}
		}
	}
	return false
}
