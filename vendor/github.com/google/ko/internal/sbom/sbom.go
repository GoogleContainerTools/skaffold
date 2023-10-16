// Copyright 2022 ko Build Authors All Rights Reserved.
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
	"bufio"
	"bytes"
	"fmt"
	"runtime/debug"
	"strings"
	"unicode"
)

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

// massageGoModVersion massages the output of `go version -m` into a form that
// can be consumed by ParseBuildInfo.
//
// `go version -m` adds a line at the beginning of its output, and tabs at the
// beginning of every line, that ParseBuildInfo doesn't like.
func massageGoVersionM(b []byte) ([]byte, error) {
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(b))
	if !scanner.Scan() {
		// Input was malformed, and doesn't contain any newlines (it
		// may even be empty). This seems to happen on Windows
		// (https://github.com/ko-build/ko/issues/535) and in unit tests.
		// Just proceed with an empty output for now, and SBOMs will be empty.
		// TODO: This should be an error.
		return nil, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("malformed input: %w", err)
	}
	for scanner.Scan() {
		// NOTE: debug.ParseBuildInfo relies on trailing tabs.
		line := strings.TrimLeftFunc(scanner.Text(), unicode.IsSpace)
		fmt.Fprintln(&out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
