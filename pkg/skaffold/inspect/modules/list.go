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
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
)

type moduleList struct {
	Modules []moduleEntry `json:"modules"`
}

type moduleEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsRemote bool   `json:"isRemote,omitempty"`
	IsRoot   bool   `json:"isRoot,omitempty"`
}

func PrintModulesList(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{ConfigurationFile: opts.Filename, RepoCacheDir: opts.RepoCacheDir})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}

	l := &moduleList{Modules: []moduleEntry{}}
	if err != nil {
		formatter.WriteErr(err)
		return err
	}
	for _, c := range cfgs {
		if c.Metadata.Name != "" {
			l.Modules = append(l.Modules, moduleEntry{Name: c.Metadata.Name, Path: c.SourceFile, IsRemote: c.IsRemote, IsRoot: c.IsRootConfig})
		} else if opts.ModulesOptions.IncludeAll {
			// if the `--all` flag is selected, include unnamed modules with the generated name `__config_<index>`
			// we need a generated name to disambiguate between multiple unnamed modules in the same file.
			l.Modules = append(l.Modules, moduleEntry{Name: fmt.Sprintf("__config_%d", c.SourceIndex), Path: c.SourceFile, IsRemote: c.IsRemote, IsRoot: c.IsRootConfig})
		}
	}
	return formatter.Write(l)
}
