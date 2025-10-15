// Copyright 2024 The Tessera authors. All Rights Reserved.
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

package tessera

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/client"
	"github.com/transparency-dev/tessera/internal/migrate"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

// NewMigrationTarget returns a MigrationTarget, which allows a personality to "import" a C2SP
// tlog-tiles or static-ct compliant log into a Tessera instance.
func NewMigrationTarget(ctx context.Context, d Driver, opts *MigrationOptions) (*MigrationTarget, error) {
	type migrateLifecycle interface {
		MigrationWriter(context.Context, *MigrationOptions) (migrate.MigrationWriter, LogReader, error)
	}
	lc, ok := d.(migrateLifecycle)
	if !ok {
		return nil, fmt.Errorf("driver %T does not implement MigrationTarget lifecycle", d)
	}
	mw, r, err := lc.MigrationWriter(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to init MigrationTarget lifecycle: %v", err)
	}
	return &MigrationTarget{
		writer:    mw,
		reader:    r,
		followers: opts.followers,
	}, nil
}

func NewMigrationOptions() *MigrationOptions {
	return &MigrationOptions{
		entriesPath:      layout.EntriesPath,
		bundleIDHasher:   defaultIDHasher,
		bundleLeafHasher: defaultMerkleLeafHasher,
	}
}

// MigrationOptions holds migration lifecycle settings for all storage implementations.
type MigrationOptions struct {
	// entriesPath knows how to format entry bundle paths.
	entriesPath func(n uint64, p uint8) string
	// bundleIDHasher knows how to create antispam leaf identities for entries in a serialised bundle.
	// This field's value must not be updated once configured or weird and probably unwanted antispam behaviour is likely to occur.
	bundleIDHasher func([]byte) ([][]byte, error)
	// bundleLeafHasher knows how to create Merkle leaf hashes for the entries in a serialised bundle.
	// This field's value must not be updated once configured or weird and probably unwanted integration behaviour is likely to occur.
	bundleLeafHasher func([]byte) ([][]byte, error)
	followers        []Follower
}

func (o MigrationOptions) EntriesPath() func(uint64, uint8) string {
	return o.entriesPath
}

func (o *MigrationOptions) LeafHasher() func([]byte) ([][]byte, error) {
	return o.bundleLeafHasher
}

// WithAntispam configures the migration target to *populate* the provided antispam storage using
// the data being migrated into the target tree.
//
// Note that since the tree is being _migrated_, the resulting target tree must match the structure
// of the source tree and so no attempt is made to reject/deduplicate entries.
func (o *MigrationOptions) WithAntispam(as Antispam) *MigrationOptions {
	if as != nil {
		o.followers = append(o.followers, as.Follower(o.bundleIDHasher))
	}
	return o
}

// MigrationTarget handles the process of migrating/importing a source log into a Tessera instance.
type MigrationTarget struct {
	writer    migrate.MigrationWriter
	reader    LogReader
	followers []Follower
}

// Migrate performs the work of importing a source log into the local Tessera instance.
//
// Any entry bundles implied by the provided source log size which are not already present in the local log
// will be fetched using the provided getEntries function, and stored by the underlying driver.
// A background process will continuously attempt to integrate these bundles into the local tree.
//
// An error will be returned if there is an unrecoverable problem encountered during the migration
// process, or if, once all entries have been copied and integrated into the local tree, the local
// root hash does not match the provided sourceRoot.
func (mt *MigrationTarget) Migrate(ctx context.Context, numWorkers uint, sourceSize uint64, sourceRoot []byte, getEntries client.EntryBundleFetcherFunc) error {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := newCopier(numWorkers, mt.writer.SetEntryBundle, getEntries)

	fromSize, err := mt.writer.IntegratedSize(ctx)
	if err != nil {
		return fmt.Errorf("fetching integrated size failed: %v", err)
	}
	c.bundlesCopied.Store(fromSize / layout.EntryBundleWidth)

	// Print stats
	go func() {
		bundlesToCopy := (sourceSize / layout.EntryBundleWidth)
		if bundlesToCopy == 0 {
			return
		}
		for {
			select {
			case <-cctx.Done():
				return
			case <-time.After(time.Second):
			}
			s, err := mt.writer.IntegratedSize(ctx)
			if err != nil {
				klog.Warningf("Size: %v", err)
			}

			info := []string{}
			bn := c.BundlesCopied()
			info = append(info, progress("copy", bn, bundlesToCopy))
			info = append(info, progress("integration", s, sourceSize))
			for _, f := range mt.followers {
				p, err := f.EntriesProcessed(ctx)
				if err != nil {
					klog.Infof("%s EntriesProcessed(): %v", f.Name(), err)
					continue
				}
				info = append(info, progress(f.Name(), p, sourceSize))
			}
			klog.Infof("Progress: %s", strings.Join(info, ", "))

		}
	}()

	// go integrate
	errG := errgroup.Group{}
	errG.Go(func() error {
		return c.Copy(cctx, fromSize, sourceSize)
	})

	var calculatedRoot []byte
	errG.Go(func() error {
		r, err := mt.writer.AwaitIntegration(cctx, sourceSize)
		if err != nil {
			return fmt.Errorf("awaiting integration failed: %v", err)
		}
		calculatedRoot = r
		return nil
	})

	for _, f := range mt.followers {
		klog.Infof("Starting %s follower", f.Name())
		go f.Follow(cctx, mt.reader)
		errG.Go(awaitFollower(cctx, f, sourceSize))
	}

	if err := errG.Wait(); err != nil {
		return fmt.Errorf("migrate failed: %v", err)
	}

	if !bytes.Equal(calculatedRoot, sourceRoot) {
		return fmt.Errorf("migration completed, but local root hash %x != source root hash %x", calculatedRoot, sourceRoot)
	}

	klog.Infof("Migration successful.")
	return nil
}

// awaitFollower returns a function which will block until the provided follower has processed
// at least as far as the provided index.
func awaitFollower(ctx context.Context, f Follower, i uint64) func() error {
	return func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}

			pos, err := f.EntriesProcessed(ctx)
			if err != nil {
				klog.Infof("%s EntriesProcessed(): %v", f.Name(), err)
				continue
			}
			if pos >= i {
				klog.Infof("%s follower complete", f.Name())
				return nil
			}
		}
	}
}

func progress(n string, p, total uint64) string {
	return fmt.Sprintf("%s: %d (%.2f%%)", n, p, (float64(p*100) / float64(total)))
}
