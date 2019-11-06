// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package licenses

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/tools/go/packages"
)

// Library is a collection of packages covered by the same license file.
type Library struct {
	// LicensePath is the path of the file containing the library's license.
	LicensePath string
	// Packages contains import paths for Go packages in this library.
	// It may not be the complete set of all packages in the library.
	Packages []string
}

// PackagesError aggregates all Packages[].Errors into a single error.
type PackagesError struct {
	pkgs []*packages.Package
}

func (e PackagesError) Error() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("errors for %q:", e.pkgs))
	packages.Visit(e.pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			str.WriteString(fmt.Sprintf("\n%s: %s", pkg.PkgPath, err))
		}
	})
	return str.String()
}

// Libraries returns the collection of libraries used by this package, directly or transitively.
// A library is a collection of one or more packages covered by the same license file.
// Packages not covered by a license will be returned as individual libraries.
// Standard library packages will be ignored.
func Libraries(ctx context.Context, importPaths ...string) ([]*Library, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode:    packages.NeedImports | packages.NeedDeps | packages.NeedFiles | packages.NeedName,
	}

	rootPkgs, err := packages.Load(cfg, importPaths...)
	if err != nil {
		return nil, err
	}

	pkgs := map[string]*packages.Package{}
	pkgsByLicense := make(map[string][]*packages.Package)
	errorOccurred := false
	packages.Visit(rootPkgs, func(p *packages.Package) bool {
		if len(p.Errors) > 0 {
			errorOccurred = true
			return false
		}
		if isStdLib(p) {
			// No license requirements for the Go standard library.
			return false
		}
		if len(p.OtherFiles) > 0 {
			glog.Warningf("%q contains non-Go code that can't be inspected for further dependencies:\n%s", p.PkgPath, strings.Join(p.OtherFiles, "\n"))
		}
		var pkgDir string
		switch {
		case len(p.GoFiles) > 0:
			pkgDir = filepath.Dir(p.GoFiles[0])
		case len(p.CompiledGoFiles) > 0:
			pkgDir = filepath.Dir(p.CompiledGoFiles[0])
		case len(p.OtherFiles) > 0:
			pkgDir = filepath.Dir(p.OtherFiles[0])
		default:
			// This package is empty - nothing to do.
			return true
		}
		licensePath, err := Find(pkgDir)
		if err != nil {
			glog.Errorf("Failed to find license for %s: %v", p.PkgPath, err)
		}
		pkgs[p.PkgPath] = p
		pkgsByLicense[licensePath] = append(pkgsByLicense[licensePath], p)
		return true
	}, nil)
	if errorOccurred {
		return nil, PackagesError{
			pkgs: rootPkgs,
		}
	}

	var libraries []*Library
	for licensePath, pkgs := range pkgsByLicense {
		if licensePath == "" {
			// No license for these packages - return each one as a separate library.
			for _, p := range pkgs {
				libraries = append(libraries, &Library{
					Packages: []string{p.PkgPath},
				})
			}
			continue
		}
		lib := &Library{
			LicensePath: licensePath,
		}
		for _, pkg := range pkgs {
			lib.Packages = append(lib.Packages, pkg.PkgPath)
		}
		libraries = append(libraries, lib)
	}
	return libraries, nil
}

// Name is the common prefix of the import paths for all of the packages in this library.
func (l *Library) Name() string {
	return commonAncestor(l.Packages)
}

func commonAncestor(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return paths[0]
	}
	sort.Strings(paths)
	min, max := paths[0], paths[len(paths)-1]
	lastSlashIndex := 0
	for i := 0; i < len(min) && i < len(max); i++ {
		if min[i] != max[i] {
			return min[:lastSlashIndex]
		}
		if min[i] == '/' {
			lastSlashIndex = i
		}
	}
	return min
}

func (l *Library) String() string {
	return l.Name()
}

// isStdLib returns true if this package is part of the Go standard library.
func isStdLib(pkg *packages.Package) bool {
	if len(pkg.GoFiles) == 0 {
		return false
	}
	return strings.HasPrefix(pkg.GoFiles[0], build.Default.GOROOT)
}
