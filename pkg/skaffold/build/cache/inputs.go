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

package cache

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// metaInputPrefix marks the hash inputs that are not dependency files: the artifact's
// configuration, its build args and its platforms. "!" sorts before the characters a path
// realistically starts with, so these entries group together at the top of a snapshot.
const metaInputPrefix = "!"

// maxSnapshotLine bounds the line length accepted when reading a snapshot back. Keys are
// file paths, so the default bufio limit is generous already, but a corrupt or truncated
// file should not make the reader allocate without bound.
const maxSnapshotLine = 1024 * 1024

// inputChanges is the difference between an artifact's hash inputs on this run and on the
// previous recorded run.
type inputChanges struct {
	modified []string
	added    []string
	removed  []string
}

// empty reports whether nothing at all changed.
func (c inputChanges) empty() bool {
	return len(c.modified) == 0 && len(c.added) == 0 && len(c.removed) == 0
}

// lines renders the changes one per line, prefixed with "~" for modified, "+" for added and
// "-" for removed, in that order.
func (c inputChanges) lines() []string {
	lines := make([]string, 0, len(c.modified)+len(c.added)+len(c.removed))
	for _, p := range c.modified {
		lines = append(lines, "~ "+displayInput(p))
	}
	for _, p := range c.added {
		lines = append(lines, "+ "+displayInput(p))
	}
	for _, p := range c.removed {
		lines = append(lines, "- "+displayInput(p))
	}
	return lines
}

// inputRecorder snapshots the inputs that feed each artifact's cache hash and diffs them
// against the previous run, so that a cache miss can name what actually changed. It is
// created per cache lookup pass, and is a no-op when nil, which is the case unless the user
// asked for it with --debug-cache.
type inputRecorder struct {
	dir string

	mu      sync.Mutex
	changes map[string]inputChanges
}

// newInputRecorder returns a recorder writing snapshots to dir, or nil if dir is empty.
func newInputRecorder(dir string) *inputRecorder {
	if dir == "" {
		return nil
	}
	return &inputRecorder{
		dir:     dir,
		changes: map[string]inputChanges{},
	}
}

// record diffs an artifact's hash inputs against its previous snapshot and stores the result
// for changesFor, then writes the new snapshot. Values are hashed, never stored verbatim:
// build args and artifact configuration routinely hold credentials, and a snapshot only ever
// needs to answer "is this the same as last time".
//
// Failures are logged and otherwise ignored: this is diagnostic bookkeeping, and must never
// be able to fail a build.
func (r *inputRecorder) record(ctx context.Context, imageName string, inputs map[string]string) {
	if r == nil {
		return
	}

	digests := make(map[string]string, len(inputs))
	for key, value := range inputs {
		if strings.ContainsAny(key, "\t\n") {
			// The snapshot format is line and tab delimited; such a key cannot round-trip.
			log.Entry(ctx).Debugf("cache input %q for %q cannot be recorded, skipping", key, imageName)
			continue
		}
		digests[key] = digestOf(value)
	}

	path := snapshotPath(r.dir, imageName)

	previous, err := readSnapshot(path)
	switch {
	case os.IsNotExist(err):
		// No previous run to compare against; report no changes rather than reporting
		// every input as newly added.
	case err != nil:
		log.Entry(ctx).Warnf("could not read cache input snapshot %s: %v", path, err)
	default:
		r.mu.Lock()
		r.changes[imageName] = diffInputs(previous, digests)
		r.mu.Unlock()
	}

	if err := writeSnapshot(path, digests); err != nil {
		log.Entry(ctx).Warnf("could not write cache input snapshot %s: %v", path, err)
	}
}

// changesFor returns what changed for an artifact since the previous run. The second result
// is false when nothing was recorded, either because recording is off or because there was
// no previous snapshot to compare against.
func (r *inputRecorder) changesFor(imageName string) (inputChanges, bool) {
	if r == nil {
		return inputChanges{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	changes, ok := r.changes[imageName]
	return changes, ok
}

// diffInputs compares two sets of input digests.
func diffInputs(previous, current map[string]string) inputChanges {
	var changes inputChanges
	for key, digest := range current {
		switch previousDigest, existed := previous[key]; {
		case !existed:
			changes.added = append(changes.added, key)
		case previousDigest != digest:
			changes.modified = append(changes.modified, key)
		}
	}
	for key := range previous {
		if _, stillPresent := current[key]; !stillPresent {
			changes.removed = append(changes.removed, key)
		}
	}

	sort.Strings(changes.modified)
	sort.Strings(changes.added)
	sort.Strings(changes.removed)
	return changes
}

// digestOf hashes an input value so that no build arg or configuration value is written to
// disk in the clear.
func digestOf(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// inputKey is the key a dependency is recorded under: its path relative to the artifact's
// workspace where that is possible. Dependency paths reach us however the builder resolved
// them, which for some configurations is absolute. Recording them as given would make the
// same file, in two checkouts of the same repository, compare as one path added and another
// removed -- burying the change the user is actually looking for. Relative keys also read
// better in the output.
func inputKey(workspace, dep string) string {
	if workspace == "" {
		return dep
	}
	rel, err := filepath.Rel(workspace, dep)
	if err != nil {
		// Mixing an absolute workspace with a relative dependency, or vice versa; there is
		// nothing sensible to relativise against, so record the path as given.
		return dep
	}
	return filepath.ToSlash(rel)
}

// displayInput strips the marker from a non-file input so that it reads naturally,
// e.g. "!config" becomes "config".
func displayInput(key string) string {
	return strings.TrimPrefix(key, metaInputPrefix)
}

// snapshotPath is where an artifact's snapshot lives. Image names can contain characters
// that are not valid in a filename, so they are replaced rather than escaped; a collision
// between two such names costs a spurious diff, nothing more.
func snapshotPath(dir, imageName string) string {
	safe := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, imageName)
	return filepath.Join(dir, safe+".txt")
}

// readSnapshot parses a snapshot file into a map of input key to digest. Lines that do not
// parse are skipped: a partially written snapshot should degrade into a wider diff, not an
// error.
func readSnapshot(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	snapshot := map[string]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, maxSnapshotLine)
	for scanner.Scan() {
		key, digest, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			continue
		}
		snapshot[key] = digest
	}
	return snapshot, scanner.Err()
}

// writeSnapshot writes the snapshot through a temporary file so that a concurrent reader,
// or a run interrupted mid-write, never sees a half-written snapshot.
func writeSnapshot(path string, digests map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	keys := make([]string, 0, len(digests))
	for key := range digests {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%s\t%s\n", key, digests[key])
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
