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

package flags

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

// BuildOutputFileFlag describes a flag which contains a BuildOutput.
type BuildOutputFileFlag struct {
	filename    string
	buildOutput BuildOutput
}

// BuildOutput is the output of `skaffold build`.
type BuildOutput struct {
	Builds []build.Artifact `json:"builds"`
}

func (t *BuildOutputFileFlag) String() string {
	return t.filename
}

// Usage Implements Usage() method for pflag interface
func (t *BuildOutputFileFlag) Usage() string {
	return "Input file with json encoded BuildOutput e.g.`skaffold build -q -o >build.out`"
}

// Set Implements Set() method for pflag interface
func (t *BuildOutputFileFlag) Set(value string) error {
	var (
		buf []byte
		err error
	)

	if value == "-" {
		buf, err = ioutil.ReadAll(os.Stdin)
	} else {
		if _, err := os.Stat(value); os.IsNotExist(err) {
			return err
		}
		buf, err = ioutil.ReadFile(value)
	}
	if err != nil {
		return err
	}

	buildOutput, err := ParseBuildOutput(buf)
	if err != nil {
		return fmt.Errorf("setting template flag: %w", err)
	}

	t.filename = value
	t.buildOutput = *buildOutput
	return nil
}

// Type Implements Type() method for pflag interface
func (t *BuildOutputFileFlag) Type() string {
	return fmt.Sprintf("%T", t)
}

// BuildArtifacts returns the Build Artifacts in the BuildOutputFileFlag
func (t *BuildOutputFileFlag) BuildArtifacts() []build.Artifact {
	return t.buildOutput.Builds
}

// NewBuildOutputFileFlag returns a new BuildOutputFile without any validation
func NewBuildOutputFileFlag(value string) *BuildOutputFileFlag {
	return &BuildOutputFileFlag{
		filename: value,
	}
}

// ParseBuildOutput parses BuildOutput from bytes
func ParseBuildOutput(b []byte) (*BuildOutput, error) {
	buildOutput := &BuildOutput{}
	if err := json.Unmarshal(b, buildOutput); err != nil {
		return nil, err
	}
	return buildOutput, nil
}
