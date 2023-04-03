//go:build !go1.18
// +build !go1.18

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

// TODO: Most of this is copied from:
// https://cs.opensource.google/go/go/+/master:src/debug/buildinfo/buildinfo.go
// https://cs.opensource.google/go/go/+/master:src/runtime/debug/mod.go
// It should be replaced with runtime/buildinfo.Read on the binary file when Go 1.18 is released.

package sbom

import (
	"fmt"
	"strconv"
	"strings"
)

func modulePackageName(mod *Module) string {
	return fmt.Sprintf("SPDXRef-Package-%s-%s",
		strings.ReplaceAll(mod.Path, "/", "."),
		mod.Version)
}

func bomRef(mod *Module) string {
	return fmt.Sprintf("pkg:golang/%s@%s?type=module", mod.Path, mod.Version)
}

func goRef(mod *Module) string {
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

// BuildInfo represents the build information read from a Go binary.
// https://cs.opensource.google/go/go/+/release-branch.go1.18:src/runtime/debug/mod.go;drc=release-branch.go1.18;l=41
type BuildInfo struct {
	GoVersion string         // Version of Go that produced this binary.
	Path      string         // The main package path
	Main      Module         // The module containing the main package
	Deps      []*Module      // Module dependencies
	Settings  []BuildSetting // Other information about the build.
}

// Module represents a module.
type Module struct {
	Path    string  // module path
	Version string  // module version
	Sum     string  // checksum
	Replace *Module // replaced by this module
}

// BuildSetting describes a setting that may be used to understand how the
// binary was built. For example, VCS commit and dirty status is stored here.
type BuildSetting struct {
	// Key and Value describe the build setting.
	// Key must not contain an equals sign, space, tab, or newline.
	// Value must not contain newlines ('\n').
	Key, Value string
}

// https://cs.opensource.google/go/go/+/release-branch.go1.18:src/strings/strings.go;drc=release-branch.go1.18;l=1181
func stringsCut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// quoteKey reports whether key is required to be quoted.
func quoteKey(key string) bool {
	return len(key) == 0 || strings.ContainsAny(key, "= \t\r\n\"`")
}

// quoteValue reports whether value is required to be quoted.
func quoteValue(value string) bool {
	return strings.ContainsAny(value, " \t\r\n\"`")
}

// https://cs.opensource.google/go/go/+/release-branch.go1.18:src/runtime/debug/mod.go;drc=release-branch.go1.18;l=121
func ParseBuildInfo(data string) (bi *BuildInfo, err error) {
	lineNum := 1
	defer func() {
		if err != nil {
			err = fmt.Errorf("could not parse Go build info: line %d: %w", lineNum, err)
		}
	}()

	var (
		pathLine  = "path\t"
		modLine   = "mod\t"
		depLine   = "dep\t"
		repLine   = "=>\t"
		buildLine = "build\t"
		newline   = "\n"
		tab       = "\t"
	)

	readModuleLine := func(elem []string) (Module, error) {
		if len(elem) != 2 && len(elem) != 3 {
			return Module{}, fmt.Errorf("expected 2 or 3 columns; got %d", len(elem))
		}
		version := elem[1]
		sum := ""
		if len(elem) == 3 {
			sum = elem[2]
		}
		return Module{
			Path:    elem[0],
			Version: version,
			Sum:     sum,
		}, nil
	}

	bi = new(BuildInfo)
	var (
		last *Module
		line string
		ok   bool
	)
	// Reverse of BuildInfo.String(), except for go version.
	for len(data) > 0 {
		line, data, ok = stringsCut(data, newline)
		if !ok {
			break
		}
		switch {
		case strings.HasPrefix(line, pathLine):
			elem := line[len(pathLine):]
			bi.Path = string(elem)
		case strings.HasPrefix(line, modLine):
			elem := strings.Split(line[len(modLine):], tab)
			last = &bi.Main
			*last, err = readModuleLine(elem)
			if err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, depLine):
			elem := strings.Split(line[len(depLine):], tab)
			last = new(Module)
			bi.Deps = append(bi.Deps, last)
			*last, err = readModuleLine(elem)
			if err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, repLine):
			elem := strings.Split(line[len(repLine):], tab)
			if len(elem) != 3 {
				return nil, fmt.Errorf("expected 3 columns for replacement; got %d", len(elem))
			}
			if last == nil {
				return nil, fmt.Errorf("replacement with no module on previous line")
			}
			last.Replace = &Module{
				Path:    string(elem[0]),
				Version: string(elem[1]),
				Sum:     string(elem[2]),
			}
			last = nil
		case strings.HasPrefix(line, buildLine):
			kv := line[len(buildLine):]
			if len(kv) < 1 {
				return nil, fmt.Errorf("build line missing '='")
			}

			var key, rawValue string
			switch kv[0] {
			case '=':
				return nil, fmt.Errorf("build line with missing key")

			case '`', '"':
				rawKey, err := strconv.QuotedPrefix(kv)
				if err != nil {
					return nil, fmt.Errorf("invalid quoted key in build line")
				}
				if len(kv) == len(rawKey) {
					return nil, fmt.Errorf("build line missing '=' after quoted key")
				}
				if c := kv[len(rawKey)]; c != '=' {
					return nil, fmt.Errorf("unexpected character after quoted key: %q", c)
				}
				key, _ = strconv.Unquote(rawKey)
				rawValue = kv[len(rawKey)+1:]

			default:
				var ok bool
				key, rawValue, ok = stringsCut(kv, "=")
				if !ok {
					return nil, fmt.Errorf("build line missing '=' after key")
				}
				if quoteKey(key) {
					return nil, fmt.Errorf("unquoted key %q must be quoted", key)
				}
			}

			var value string
			if len(rawValue) > 0 {
				switch rawValue[0] {
				case '`', '"':
					var err error
					value, err = strconv.Unquote(rawValue)
					if err != nil {
						return nil, fmt.Errorf("invalid quoted value in build line")
					}

				default:
					value = rawValue
					if quoteValue(value) {
						return nil, fmt.Errorf("unquoted value %q must be quoted", value)
					}
				}
			}

			bi.Settings = append(bi.Settings, BuildSetting{Key: key, Value: value})
		}
		lineNum++
	}
	return bi, nil
}
