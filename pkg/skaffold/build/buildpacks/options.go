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

package buildpacks

import "github.com/buildpacks/pack"

var buildArgsForDebug = map[string]string{
	"GOOGLE_GOGCFLAGS": "'all=-N -l'", // disable build optimization for Golang
	// TODO: Add for other languages
}

var buildArgsForDev = map[string]string{
	"GOOGLE_GOLDFLAGS": "-w", // omit debug information in build output for Golang
	// TODO: Add for other languages
}

type BuildOptionsModifier func(*pack.BuildOptions) *pack.BuildOptions

type BuildOptions struct {
	Tag       string
	modifiers []BuildOptionsModifier
}

func (b *BuildOptions) AddModifier(m BuildOptionsModifier) *BuildOptions {
	b.modifiers = append(b.modifiers, m)
	return b
}

func (b *BuildOptions) ApplyModifiers(opts *pack.BuildOptions) *pack.BuildOptions {
	for _, modifier := range b.modifiers {
		opts = modifier(opts)
	}
	return opts
}

func OptimizeBuildForDebug(opts *pack.BuildOptions) *pack.BuildOptions {
	if opts == nil {
		opts = &pack.BuildOptions{}
	}

	if opts.Env == nil {
		opts.Env = make(map[string]string)
	}

	for k, v := range buildArgsForDebug {
		if _, exists := opts.Env[k]; !exists {
			opts.Env[k] = v
		}
	}
	return opts
}

func OptimizeBuildForDev(opts *pack.BuildOptions) *pack.BuildOptions {
	if opts == nil {
		opts = &pack.BuildOptions{}
	}

	if opts.Env == nil {
		opts.Env = make(map[string]string)
	}

	for k, v := range buildArgsForDev {
		if _, exists := opts.Env[k]; !exists {
			opts.Env[k] = v
		}
	}
	return opts
}
