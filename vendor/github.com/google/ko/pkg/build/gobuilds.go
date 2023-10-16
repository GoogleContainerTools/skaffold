// Copyright 2021 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import (
	"context"
	"fmt"
	gb "go/build"
	"path"
	"path/filepath"
	"strings"
)

type gobuilds struct {
	// Map of fully qualified import path to go builder with config
	builders map[string]builderWithConfig

	// Default go builder used if there's no matching build config.
	defaultBuilder Interface

	// workingDirectory is typically ".", but it may be a different value if ko is embedded as a library.
	workingDirectory string
}

// builderWithConfig is not an imaginative name.
type builderWithConfig struct {
	builder Interface
	config  Config
}

// NewGobuilds returns a build.Interface that can dispatch to builders based on matching the import path to a build config in .ko.yaml.
func NewGobuilds(ctx context.Context, workingDirectory string, buildConfigs map[string]Config, opts ...Option) (Interface, error) {
	if workingDirectory == "" {
		workingDirectory = "."
	}
	defaultBuilder, err := NewGo(ctx, workingDirectory, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not create default go builder: %w", err)
	}
	g := &gobuilds{
		builders:         map[string]builderWithConfig{},
		defaultBuilder:   defaultBuilder,
		workingDirectory: workingDirectory,
	}
	for importpath, buildConfig := range buildConfigs {
		builderDirectory := path.Join(workingDirectory, buildConfig.Dir)
		builder, err := NewGo(ctx, builderDirectory, opts...)
		if err != nil {
			return nil, fmt.Errorf("could not create go builder for config (%q): %w", importpath, err)
		}
		g.builders[importpath] = builderWithConfig{
			builder: builder,
			config:  buildConfig,
		}
	}
	return g, nil
}

// QualifyImport implements build.Interface
func (g *gobuilds) QualifyImport(importpath string) (string, error) {
	b := g.builder(importpath)
	if b.config.Dir != "" {
		var err error
		importpath, err = relativePath(b.config.Dir, importpath)
		if err != nil {
			return "", err
		}
	}
	return b.builder.QualifyImport(importpath)
}

// IsSupportedReference implements build.Interface
func (g *gobuilds) IsSupportedReference(importpath string) error {
	return g.builder(importpath).builder.IsSupportedReference(importpath)
}

// Build implements build.Interface
func (g *gobuilds) Build(ctx context.Context, importpath string) (Result, error) {
	return g.builder(importpath).builder.Build(ctx, importpath)
}

// builder selects a go builder for the provided import path.
// The `importpath` argument can be either local (e.g., `./cmd/foo`) or not (e.g., `example.com/app/cmd/foo`).
func (g *gobuilds) builder(importpath string) builderWithConfig {
	importpath = strings.TrimPrefix(importpath, StrictScheme)
	if len(g.builders) == 0 {
		return builderWithConfig{
			builder: g.defaultBuilder,
		}
	}
	// first, try to find go builder by fully qualified import path
	if builderWithConfig, exists := g.builders[importpath]; exists {
		return builderWithConfig
	}
	// second, try to find go builder by local path
	for _, builderWithConfig := range g.builders {
		// Match go builder by trying to resolve the local path to a fully qualified import path. If successful, we have a winner.
		relPath, err := relativePath(builderWithConfig.config.Dir, importpath)
		if err != nil {
			// Cannot determine a relative path. Move on and try the next go builder.
			continue
		}
		_, err = builderWithConfig.builder.QualifyImport(relPath)
		if err != nil {
			// There's an error turning the local path into a fully qualified import path. Move on and try the next go builder.
			continue
		}
		return builderWithConfig
	}
	// fall back to default go builder
	return builderWithConfig{
		builder: g.defaultBuilder,
	}
}

// relativePath takes as input a local import path, and returns a path relative to the base directory.
//
// For example, given the following inputs:
// - baseDir: "app"
// - importpath: "./app/cmd/foo
// The output is: "./cmd/foo"
//
// If the input is a not a local import path as determined by go/build.IsLocalImport(), the input is returned unchanged.
//
// If the import path is _not_ a subdirectory of baseDir, the result is an error.
func relativePath(baseDir string, importpath string) (string, error) {
	// Return input unchanged if the import path is a fully qualified import path
	if !gb.IsLocalImport(importpath) {
		return importpath, nil
	}
	relPath, err := filepath.Rel(baseDir, importpath)
	if err != nil {
		return "", fmt.Errorf("cannot determine relative path of baseDir (%q) and local path (%q): %w", baseDir, importpath, err)
	}
	if strings.HasPrefix(relPath, "..") {
		// TODO Is this assumption correct?
		return "", fmt.Errorf("import path (%q) must be a subdirectory of build config directory (%q)", importpath, baseDir)
	}
	if !strings.HasPrefix(relPath, ".") && relPath != "." {
		relPath = "./" + relPath // ensure go/build.IsLocalImport() interprets this as a local path
	}
	return relPath, nil
}
