/*
Copyright 2026 The Skaffold Authors

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

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/moby/patternmatcher"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestWalkWorkspaceSkipsIgnoredDirectories(t *testing.T) {
	files := []string{
		"app/src/main.js",
		"app/node_modules/pkg/index.js",
		"app/node_modules/pkg/README.md",
		"vendor/kept/lib/a.php",
		"vendor/kept/node_modules/b.js",
		"vendor/other/lib/c.php",
	}
	tests := []struct {
		description string
		excludes    []string
		// expectedDeps are the files that are not ignored.
		expectedDeps []string
		// notVisited are directories whose entries must not be read.
		notVisited []string
	}{
		{
			description:  "no negated pattern",
			excludes:     []string{"**/node_modules", "vendor/other"},
			expectedDeps: []string{"app/src/main.js", "vendor/kept/lib/a.php"},
			notVisited:   []string{"app/node_modules", "vendor/kept/node_modules", "vendor/other"},
		},
		{
			description:  "negated directory below an ignored directory",
			excludes:     []string{"**/node_modules", "vendor", "!vendor/kept/lib"},
			expectedDeps: []string{"app/src/main.js", "vendor/kept/lib/a.php"},
			notVisited:   []string{"app/node_modules", "vendor/kept/node_modules", "vendor/other"},
		},
		{
			description:  "negated pattern that ends with a wildcard",
			excludes:     []string{"**/node_modules", "vendor/*", "!vendor/kept/**"},
			expectedDeps: []string{"app/src/main.js", "vendor/kept/lib/a.php", "vendor/kept/node_modules/b.js"},
			notVisited:   []string{"app/node_modules", "vendor/other"},
		},
		{
			description:  "pattern after the negated pattern ignores again",
			excludes:     []string{"vendor/*", "!vendor/kept/**", "**/node_modules"},
			expectedDeps: []string{"app/src/main.js", "vendor/kept/lib/a.php"},
			notVisited:   []string{"app/node_modules", "vendor/kept/node_modules", "vendor/other"},
		},
		{
			description:  "negated pattern with a wildcard for the directory",
			excludes:     []string{"**/node_modules", "vendor", "!*/node_modules/pkg/index.js"},
			expectedDeps: []string{"app/node_modules/pkg/index.js", "app/src/main.js"},
			notVisited:   []string{"vendor/kept", "vendor/other"},
		},
		{
			description:  "negated pattern that matches at any depth",
			excludes:     []string{"**/node_modules", "vendor/other", "!**/*.md"},
			expectedDeps: []string{"app/node_modules/pkg/README.md", "app/src/main.js", "vendor/kept/lib/a.php"},
		},
		{
			description:  "negated pattern with the name of an ignored file",
			excludes:     []string{"**", "!main.js", "!app/src"},
			expectedDeps: []string{"app/src/main.js"},
			notVisited:   []string{"app/node_modules", "vendor"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			workspace := t.NewTempDir().Touch(files...).Root()

			deps, err := WalkWorkspace(workspace, test.excludes, []string{"."})
			t.CheckNoError(err)
			var actualDeps []string
			for dep := range deps {
				actualDeps = append(actualDeps, filepath.ToSlash(dep))
			}
			sort.Strings(actualDeps)
			t.CheckDeepEqual(test.expectedDeps, actualDeps)

			ignored, err := NewDockerIgnorePredicate(workspace, test.excludes)
			t.CheckNoError(err)
			visited := map[string]bool{}
			err = walk.From(workspace).When(func(path string, info walk.Dirent) (bool, error) {
				if path == workspace {
					return false, nil
				}
				visited[filepath.ToSlash(filepath.Dir(strings.TrimPrefix(path, workspace+string(filepath.Separator))))] = true
				return ignored(path, info)
			}).Do(func(string, walk.Dirent) error { return nil })
			t.CheckNoError(err)
			for _, dir := range test.notVisited {
				if visited[dir] {
					t.Errorf("entries of ignored directory %q were visited", dir)
				}
			}
		})
	}
}

func TestWalkWorkspaceOverlappingDependencies(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("app/src/main.js", "app/src/lib/util.js", "app/node_modules/pkg/index.js", "app/package.json")
		tmpDir.Mkdir("app/empty").Mkdir("app/cache")
		excludes := []string{"**/node_modules", "app/cache", "!app/cache/keep"}

		// A directory that is ignored as part of "app" is still walked when it is a dependency itself.
		deps, err := WalkWorkspace(tmpDir.Root(), excludes, []string{"app", "app/src", "app/package.json", "app/src/lib", "app/cache", "app/node_modules", "app"})
		t.CheckNoError(err)

		var actual []string
		for dep := range deps {
			actual = append(actual, filepath.ToSlash(dep))
		}
		sort.Strings(actual)
		t.CheckDeepEqual([]string{"app/cache", "app/empty", "app/package.json", "app/src/lib/util.js", "app/src/main.js"}, actual)
	})
}

// TestDockerIgnoreMatchesLikePatternMatcher compares dockerIgnore with patternmatcher,
// which Skaffold used before, on random patterns and paths.
func TestDockerIgnoreMatchesLikePatternMatcher(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	names := []string{"a", "b", "ab", "c.md", "node_modules"}
	segments := append([]string{"*", "**", "a*", "*.md", "?", "[ab]", "!b"}, names...)

	sep := string(filepath.Separator)
	var paths []string
	var addPaths func(dir string, depth int)
	addPaths = func(dir string, depth int) {
		for _, name := range names {
			paths = append(paths, dir+name)
			if depth < 4 {
				addPaths(dir+name+sep, depth+1)
			}
		}
	}
	addPaths("", 1)

	for n := 0; n < 300; n++ {
		var excludes []string
		for i := rnd.Intn(6) + 1; i > 0; i-- {
			var pattern []string
			for j := rnd.Intn(4) + 1; j > 0; j-- {
				pattern = append(pattern, segments[rnd.Intn(len(segments))])
			}
			exclude := strings.Join(pattern, "/")
			if rnd.Intn(3) == 0 {
				exclude = "!" + exclude
			}
			excludes = append(excludes, exclude)
		}

		expected, err := patternmatcher.New(excludes)
		testutil.CheckError(t, false, err)
		actual, err := newDockerIgnore(excludes)
		testutil.CheckError(t, false, err)

		// Paths are sorted, so that a directory comes before the paths below it.
		skipped := ""
		for _, path := range paths {
			expectedIgnored, err := expected.MatchesOrParentMatches(path)
			testutil.CheckError(t, false, err)

			if skipped != "" && strings.HasPrefix(path, skipped) {
				if !expectedIgnored {
					t.Fatalf("patterns %q: %q is not ignored, but %q was skipped", excludes, path, skipped)
				}
				continue
			}

			// Every path is asked about both as a file and as a directory.
			fileIgnored, err := actual.fileIgnored(path)
			testutil.CheckError(t, false, err)
			dir, err := actual.dir(path)
			testutil.CheckError(t, false, err)
			if fileIgnored != expectedIgnored || dir.ignored != expectedIgnored {
				t.Fatalf("patterns %q, path %q: expected ignored %v, got %v as a file and %v as a directory", excludes, path, expectedIgnored, fileIgnored, dir.ignored)
			}
			if dir.ignored && actual.canSkip(path, dir) {
				skipped = path + sep
			}
		}
	}
}

// BenchmarkWalkWorkspace lists the dependencies of a Dockerfile with 151 COPY
// instructions in a workspace where most files are ignored.
func BenchmarkWalkWorkspace(b *testing.B) {
	workspace := b.TempDir()
	touch := func(format string, args ...interface{}) {
		path := filepath.Join(workspace, filepath.FromSlash(fmt.Sprintf(format, args...)))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			b.Fatal(err)
		}
	}

	var deps []string
	for s := 0; s < 30; s++ {
		svc := fmt.Sprintf("services/svc%02d", s)
		touch("%s/package.json", svc)
		touch("%s/tsconfig.json", svc)
		for d := 0; d < 8; d++ {
			for f := 0; f < 25; f++ {
				touch("%s/src/mod%d/sub/deep/file%d.ts", svc, d, f)
			}
		}
		for d := 0; d < 4; d++ {
			for f := 0; f < 10; f++ {
				touch("%s/config/env%d/c%d.yaml", svc, d, f)
			}
		}
		for p := 0; p < 40; p++ {
			for f := 0; f < 25; f++ {
				touch("%s/node_modules/pkg%d/lib/f%d.js", svc, p, f)
			}
		}
		touch("%s/build/out.log", svc)
		deps = append(deps, svc+"/package.json", svc+"/tsconfig.json", svc+"/src", svc+"/config", svc)
	}
	for d := 0; d < 20; d++ {
		for f := 0; f < 25; f++ {
			touch("shopware/deps/pcs/Bundle%d/src/F%d.php", d, f)
			touch("shopware/deps/other/Bundle%d/src/F%d.php", d, f)
			touch("shopware/deps/pcs/node_modules/p%d/f%d.js", d, f)
		}
	}
	deps = append(deps, "shopware")

	excludes := []string{
		"**/node_modules", "shopware/deps/*", "!shopware/deps/pcs/**", "**/node_modules",
		"**/.git", "**/*.log", "**/*.tmp", "**/.DS_Store", "**/build", "**/dist", "**/coverage",
		"**/.idea", "**/.vscode", "**/*.swp", "**/__pycache__", "**/*.pyc", "tmp", "docs", "**/*.md",
		"!README.md", "services/*/test", "services/*/tests", "services/*/.env*", "**/.cache",
		"**/var/cache", "**/var/log", "**/*.orig", "**/*.rej", "**/Thumbs.db", "**/.terraform",
		"**/*.tfstate", "**/npm-debug.log*", "**/yarn-error.log", "**/.eslintcache", "**/.phpunit*",
		"**/phpstan-baseline.neon", "infrastructure", "scripts/local", "**/docker-compose*.yml",
		"**/Dockerfile.*", "**/*.bak", "**/.gitignore", "**/.gitattributes", "k8s", "**/*.test.snap",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := WalkWorkspace(workspace, excludes, deps)
		if err != nil {
			b.Fatal(err)
		}
		// 30 services with 242 files each, and the 500 files of shopware/deps/pcs.
		if len(files) != 30*242+500 {
			b.Fatalf("unexpected number of dependencies: %d", len(files))
		}
	}
}
