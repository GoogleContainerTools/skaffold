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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

var (
	ReadFileFunc  = util.ReadConfiguration
	WriteFileFunc = func(filename string, data []byte) error {
		return ioutil.WriteFile(filename, data, 0644)
	}
)

// MarshalConfigSet marshals out the slice of skaffold configs into the respective source `skaffold.yaml` files.
// It ensures that the unmodified configs are copied over as-is in their original positions in the file.
func MarshalConfigSet(cfgs parser.SkaffoldConfigSet) error {
	m := make(map[string]parser.SkaffoldConfigSet)
	for _, cfg := range cfgs {
		m[cfg.SourceFile] = append(m[cfg.SourceFile], cfg)
	}
	for file, set := range m {
		if err := marshalConfigSetForFile(file, set); err != nil {
			return err
		}
	}
	return nil
}

func marshalConfigSetForFile(filename string, cfgs parser.SkaffoldConfigSet) error {
	buf, err := ReadFileFunc(filename)
	if err != nil {
		return sErrors.ConfigParsingError(err)
	}
	in := bytes.NewReader(buf)
	decoder := yamlv3.NewDecoder(in)
	decoder.KnownFields(true)
	var sl []interface{}
	for {
		var parsed yamlv3.Node
		err := decoder.Decode(&parsed)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to parse YAML: %w", err)
		}
		// parsed content is a document so the `Content` slice has exactly one element
		sl = append(sl, parsed.Content[0])
	}

	for i, cfg := range cfgs {
		sl[cfg.SourceIndex] = cfgs[i].SkaffoldConfig
	}

	newCfgs, err := yaml.MarshalWithSeparator(sl)
	if err != nil {
		return fmt.Errorf("marshaling new configs: %w", err)
	}
	if err := WriteFileFunc(filename, newCfgs); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
