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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
)

type testList struct {
	Tests []interface{} `json:"tests"`
}

// CustomTest entries are handled by CustomTest struct, there is no StructureTest so structureTestEntry is required
type structureTestEntry struct {
	StructureTest     string   `json:"structureTest"`
	StructureTestArgs []string `json:"structureTestArgs"`
}

func PrintTestsList(ctx context.Context, out io.Writer, opts inspect.Options) error {
	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.TestsProfiles,
	})
	if err != nil {
		return formatter.WriteErr(err)
	}

	l := &testList{Tests: []interface{}{}}
	for _, c := range cfgs {
		for _, t := range c.Test {
			for _, ct := range t.CustomTests {
				l.Tests = append(l.Tests, ct)
			}
			for _, st := range t.StructureTests {
				l.Tests = append(l.Tests, structureTestEntry{StructureTest: st, StructureTestArgs: t.StructureTestArgs})
			}
		}
	}
	return formatter.Write(l)
}
