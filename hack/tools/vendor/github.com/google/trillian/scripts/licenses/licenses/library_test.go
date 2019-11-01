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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestLibraries(t *testing.T) {
	for _, test := range []struct {
		desc       string
		importPath string
		wantLibs   []string
	}{
		{
			desc:       "Detects direct dependency",
			importPath: "github.com/google/trillian/scripts/licenses/licenses/testdata/direct",
			wantLibs: []string{
				"github.com/google/trillian/scripts/licenses/licenses/testdata/direct",
				"github.com/google/trillian/scripts/licenses/licenses/testdata/indirect",
			},
		},
		{
			desc:       "Detects transitive dependency",
			importPath: "github.com/google/trillian/scripts/licenses/licenses/testdata",
			wantLibs: []string{
				"github.com/google/trillian/scripts/licenses/licenses/testdata",
				"github.com/google/trillian/scripts/licenses/licenses/testdata/direct",
				"github.com/google/trillian/scripts/licenses/licenses/testdata/indirect",
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			gotLibs, err := Libraries(context.Background(), test.importPath)
			if err != nil {
				t.Fatalf("Libraries(_, %q) = (_, %q), want (_, nil)", test.importPath, err)
			}
			var gotLibNames []string
			for _, lib := range gotLibs {
				gotLibNames = append(gotLibNames, lib.Name())
			}
			if diff := cmp.Diff(test.wantLibs, gotLibNames, cmpopts.SortSlices(func(x, y string) bool { return x < y })); diff != "" {
				t.Errorf("Libraries(_, %q): diff (-want +got)\n%s", test.importPath, diff)
			}
		})
	}
}

func TestLibraryName(t *testing.T) {
	for _, test := range []struct {
		desc     string
		lib      *Library
		wantName string
	}{
		{
			desc:     "Library with no packages",
			lib:      &Library{},
			wantName: "",
		},
		{
			desc: "Library with 1 package",
			lib: &Library{
				Packages: []string{
					"github.com/google/trillian/crypto",
				},
			},
			wantName: "github.com/google/trillian/crypto",
		},
		{
			desc: "Library with 2 packages",
			lib: &Library{
				Packages: []string{
					"github.com/google/trillian/crypto",
					"github.com/google/trillian/server",
				},
			},
			wantName: "github.com/google/trillian",
		},
		{
			desc: "Vendored library",
			lib: &Library{
				Packages: []string{
					"github.com/google/trillian/vendor/coreos/etcd",
				},
			},
			wantName: "github.com/google/trillian/vendor/coreos/etcd",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			if got, want := test.lib.Name(), test.wantName; got != want {
				t.Fatalf("Name() = %q, want %q", got, want)
			}
		})
	}
}
