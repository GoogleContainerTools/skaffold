//
// Copyright 2025 The Sigstore Authors.
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
	"context"
	"fmt"
	"log/slog"
	"time"

	rekor_pb "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor-tiles/pkg/note"
	"github.com/sigstore/sigstore/pkg/signature"
	logformat "github.com/transparency-dev/formats/log"
	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera"
	"github.com/transparency-dev/tessera/client"
)

const (
	DefaultBatchMaxSize           = tessera.DefaultBatchMaxSize
	DefaultBatchMaxAge            = tessera.DefaultBatchMaxAge
	DefaultCheckpointInterval     = tessera.DefaultCheckpointInterval
	DefaultPushbackMaxOutstanding = tessera.DefaultPushbackMaxOutstanding
)

type DuplicateError struct {
	index uint64
}

func (e DuplicateError) Error() string {
	return fmt.Sprintf("an equivalent entry already exists in the transparency log with index %d", e.index)
}

type InclusionProofVerificationError struct {
	index uint64
	err   error
}

func (e InclusionProofVerificationError) Error() string {
	return fmt.Sprintf("verifying inclusion proof for index %d: %v", e.index, e.err)
}

// Storage provides the functions to add entries to a Tessera log.
type Storage interface {
	Add(ctx context.Context, entry *tessera.Entry) (*rekor_pb.TransparencyLogEntry, error)
	ReadTile(ctx context.Context, level, index uint64, p uint8) ([]byte, error)
}

type storage struct {
	origin     string
	awaiter    *tessera.PublicationAwaiter
	addFn      tessera.AddFn
	readTileFn client.TileFetcherFunc
}

// NewAppendOptions initializes the Tessera append options with a checkpoint signer, which is the only non-optional append option.
func NewAppendOptions(ctx context.Context, origin string, signer signature.Signer) (*tessera.AppendOptions, error) {
	opts := tessera.NewAppendOptions()
	noteSigner, err := note.NewNoteSigner(ctx, origin, signer)
	if err != nil {
		return nil, fmt.Errorf("getting note signer: %w", err)
	}
	opts = opts.WithCheckpointSigner(noteSigner)
	return opts, nil
}

// WithLifecycleOptions accepts an initialized AppendOptions and adds batching, checkpoint, and pushback options to it.
// It returns the mutated options object for readability.
func WithLifecycleOptions(opts *tessera.AppendOptions, batchMaxSize uint, baxMaxAge time.Duration, checkpointInterval time.Duration, pushback uint) *tessera.AppendOptions {
	opts = opts.WithBatching(batchMaxSize, baxMaxAge)
	opts = opts.WithCheckpointInterval(checkpointInterval)
	opts = opts.WithPushback(pushback)
	return opts
}

// WithAntispamOptions accepts an initialized AppendOptions and adds antispam options to it.
// Accepts an optional persistent antispam provider. If nil, antispam does not persist between
// server restarts. Returns the mutated options object for readability.
func WithAntispamOptions(opts *tessera.AppendOptions, as tessera.Antispam) *tessera.AppendOptions {
	inMemoryLRUSize := uint(256) // There's no documentation providing guidance on this cache size. Use a hard-coded value for now and consider exposing it as a configuration option later.
	opts = opts.WithAntispam(inMemoryLRUSize, as)
	return opts
}

// DriverConfiguration contains storage-specific configuration for each supported storage backend.
type DriverConfiguration struct {
	// GCP configuration
	GCPBucket    string
	GCPSpannerDB string

	// Antispam configuration
	PersistentAntispam  bool
	ASMaxBatchSize      uint
	ASPushbackThreshold uint
}

// NewDriver creates a Tessera driver and optional persistent antispam for a given storage backend.
func NewDriver(ctx context.Context, config DriverConfiguration) (tessera.Driver, tessera.Antispam, error) {
	switch {
	case config.GCPBucket != "" && config.GCPSpannerDB != "":
		driver, err := NewGCPDriver(ctx, config.GCPBucket, config.GCPSpannerDB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize GCP driver: %v", err.Error())
		}
		var persistentAntispam tessera.Antispam
		if config.PersistentAntispam {
			as, err := NewGCPAntispam(ctx, config.GCPSpannerDB, config.ASMaxBatchSize, config.ASPushbackThreshold)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to initialize GCP antispam: %v", err.Error())
			}
			persistentAntispam = as
		}
		return driver, persistentAntispam, nil
	default:
		return nil, nil, fmt.Errorf("no flags provided to initialize Tessera driver")
	}
}

// NewStorage creates a Tessera storage object for the provided driver and signer.
// Returns the storage object and a function that must be called when shutting down the server.
func NewStorage(ctx context.Context, origin string, driver tessera.Driver, appendOptions *tessera.AppendOptions) (Storage, func(context.Context) error, error) {
	appender, shutdown, reader, err := tessera.NewAppender(ctx, driver, appendOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("getting tessera appender: %w", err)
	}
	slog.Info("starting Tessera sequencer")
	awaiter := tessera.NewPublicationAwaiter(ctx, reader.ReadCheckpoint, 1*time.Second)
	return &storage{
		origin:     origin,
		awaiter:    awaiter,
		addFn:      appender.Add,
		readTileFn: reader.ReadTile,
	}, shutdown, nil
}

// Add adds a Tessera entry to the log, waits for it to be sequenced into the log,
// and returns the log index and inclusion proof as a TransparencyLogEntry object.
func (s *storage) Add(ctx context.Context, entry *tessera.Entry) (*rekor_pb.TransparencyLogEntry, error) {
	idx, dup, checkpointBody, err := s.addEntry(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("add entry: %w", err)
	}
	if dup {
		return nil, DuplicateError{index: idx.U()}
	}
	inclusionProof, err := s.buildProof(ctx, idx, checkpointBody, entry.LeafHash())
	if err != nil {
		return nil, fmt.Errorf("building inclusion proof: %w", err)
	}
	return &rekor_pb.TransparencyLogEntry{
		LogIndex:          idx.I(),
		InclusionProof:    inclusionProof,
		CanonicalizedBody: entry.Data(),
	}, nil
}

// ReadTile looks up the tile at the given level, index within the level, and
// width of the tile if partial, and returns the raw bytes of the tile.
func (s *storage) ReadTile(ctx context.Context, level, index uint64, p uint8) ([]byte, error) {
	tile, err := s.readTileFn(ctx, level, index, p)
	if err != nil {
		return nil, fmt.Errorf("reading tile level %d index %d p %d: %w", level, index, p, err)
	}
	return tile, nil
}

func (s *storage) addEntry(ctx context.Context, entry *tessera.Entry) (*SafeInt64, bool, []byte, error) {
	idx, checkpointBody, err := s.awaiter.Await(ctx, s.addFn(ctx, entry))
	if err != nil {
		return nil, false, nil, fmt.Errorf("await: %w", err)
	}
	safeIdx, err := NewSafeInt64(idx.Index)
	if err != nil {
		return nil, false, nil, fmt.Errorf("invalid index: %w", err)
	}
	return safeIdx, idx.IsDup, checkpointBody, nil
}

func (s *storage) buildProof(ctx context.Context, idx *SafeInt64, signedCheckpoint, leafHash []byte) (*rekor_pb.InclusionProof, error) {
	checkpoint, err := unmarshalCheckpoint(signedCheckpoint)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling checkpoint: %w", err)
	}
	proofBuilder, err := client.NewProofBuilder(ctx, checkpoint.Size, s.ReadTile)
	if err != nil {
		return nil, fmt.Errorf("new proof builder: %w", err)
	}
	inclusionProof, err := proofBuilder.InclusionProof(ctx, idx.U())
	if err != nil {
		return nil, fmt.Errorf("generating inclusion proof: %w", err)
	}
	safeCheckpointSize, err := NewSafeInt64(checkpoint.Size)
	if err != nil {
		return nil, fmt.Errorf("invalid tree size: %d", checkpoint.Size)
	}
	if err := proof.VerifyInclusion(rfc6962.DefaultHasher, idx.U(), safeCheckpointSize.U(), leafHash, inclusionProof, checkpoint.Hash); err != nil {
		return nil, InclusionProofVerificationError{idx.U(), err}
	}
	return &rekor_pb.InclusionProof{
		LogIndex: idx.I(),
		RootHash: checkpoint.Hash,
		TreeSize: safeCheckpointSize.I(),
		Hashes:   inclusionProof,
		Checkpoint: &rekor_pb.Checkpoint{
			Envelope: string(signedCheckpoint),
		},
	}, nil
}

func unmarshalCheckpoint(checkpointBody []byte) (logformat.Checkpoint, error) {
	checkpoint := logformat.Checkpoint{}
	_, err := checkpoint.Unmarshal(checkpointBody)
	return checkpoint, err
}
