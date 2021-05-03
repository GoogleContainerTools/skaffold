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
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/parser"
)

var (
	getConfigSetFunc = parser.GetConfigSet
)

type moduleList struct {
	Modules []moduleEntry `json:"modules"`
}

type moduleEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func PrintModulesList(ctx context.Context, out io.Writer, opts Options) error {
	formatter := getOutputFormatter(out, opts.OutFormat)
	cfgs, err := getConfigSetFunc(config.SkaffoldOptions{ConfigurationFile: opts.Filename})
	if err != nil {
		return formatter.WriteErr(err)
	}

	l := &moduleList{}
	for _, c := range cfgs {
		if c.Metadata.Name != "" {
			l.Modules = append(l.Modules, moduleEntry{Name: c.Metadata.Name, Path: c.SourceFile})
		}
	}
	return formatter.Write(l)
}
