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

package docker

import "github.com/docker/docker/api/types"

var buildArgsForDebug = map[string]string{
	"GO_GCFLAGS": "'all=-N -l'", // disable build optimization for Golang
	// TODO: Add for other languages
}

var buildArgsForRelease = map[string]string{
	"GO_LDFLAGS": "-w", // omit debug information in build output for Golang
	// TODO: Add for other languages
}

type BuildOptionsModifier func(*types.ImageBuildOptions) *types.ImageBuildOptions

type BuildOptions struct {
	Tag       string
	modifiers []BuildOptionsModifier
}

func (b *BuildOptions) AddModifier(m BuildOptionsModifier) *BuildOptions {
	b.modifiers = append(b.modifiers, m)
	return b
}

func (b *BuildOptions) ApplyModifiers(opts *types.ImageBuildOptions) *types.ImageBuildOptions {
	for _, modifier := range b.modifiers {
		opts = modifier(opts)
	}
	return opts
}

func OptimizeBuildForDebug(opts *types.ImageBuildOptions) *types.ImageBuildOptions {
	if opts == nil {
		opts = &types.ImageBuildOptions{}
	}

	if opts.BuildArgs == nil {
		opts.BuildArgs = make(map[string]*string)
	}

	for k, v := range buildArgsForDebug {
		if opts.BuildArgs[k] == nil {
			opts.BuildArgs[k] = &v
		}
	}

	return opts
}

func OptimizeBuildForRelease(opts *types.ImageBuildOptions) *types.ImageBuildOptions {
	if opts == nil {
		opts = &types.ImageBuildOptions{}
	}

	if opts.BuildArgs == nil {
		opts.BuildArgs = make(map[string]*string)
	}

	for k, v := range buildArgsForRelease {
		if opts.BuildArgs[k] == nil {
			opts.BuildArgs[k] = &v
		}
	}

	return opts
}
