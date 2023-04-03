//go:build go1.18
// +build go1.18

// Copyright 2021 Google LLC All Rights Reserved.
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

package sbom

import (
	"fmt"
	"runtime/debug"
	"strings"
)

type BuildInfo debug.BuildInfo

func ParseBuildInfo(data string) (*BuildInfo, error) {
	dbi, err := debug.ParseBuildInfo(data)
	if err != nil {
		return nil, fmt.Errorf("parsing build info: %w", err)
	}
	bi := BuildInfo(*dbi)
	return &bi, nil
}

func modulePackageName(mod *debug.Module) string {
	return fmt.Sprintf("SPDXRef-Package-%s-%s",
		strings.ReplaceAll(mod.Path, "/", "."),
		mod.Version)
}

func bomRef(mod *debug.Module) string {
	return fmt.Sprintf("pkg:golang/%s@%s?type=module", mod.Path, mod.Version)
}

func goRef(mod *debug.Module) string {
	path := mod.Path
	// Try to lowercase the first 2 path elements to comply with spec
	// https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#golang
	p := strings.Split(path, "/")
	if len(p) > 2 {
		path = strings.Join(
			append(
				[]string{strings.ToLower(p[0]), strings.ToLower(p[1])},
				p[2:]...,
			), "/",
		)
	}
	return fmt.Sprintf("pkg:golang/%s@%s?type=module", path, mod.Version)
}
