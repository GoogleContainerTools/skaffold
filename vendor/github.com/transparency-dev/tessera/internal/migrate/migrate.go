// Copyright 2025 The Tessera authors. All Rights Reserved.
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

// Package migrate contains internal implementations for migration.
package migrate

import "context"

type MigrationWriter interface {
	// SetEntryBundle stores the provided serialised entry bundle at the location implied by the provided
	// entry bundle index and partial size.
	//
	// Bundles may be set in any order (not just consecutively), and the implementation should integrate
	// them into the local tree in the most efficient way possible.
	//
	// Writes should be idempotent; repeated calls to set the same bundle with the same data should not
	// return an error.
	SetEntryBundle(ctx context.Context, idx uint64, partial uint8, bundle []byte) error
	// AwaitIntegration should block until the local integrated tree has grown to the provided size,
	// and should return the locally calculated root hash derived from the integration of the contents of
	// entry bundles set using SetEntryBundle above.
	AwaitIntegration(ctx context.Context, size uint64) ([]byte, error)
	// IntegratedSize returns the current size of the locally integrated log.
	IntegratedSize(ctx context.Context) (uint64, error)
}
