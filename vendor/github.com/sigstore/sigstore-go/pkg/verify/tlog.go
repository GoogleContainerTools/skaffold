// Copyright 2023 The Sigstore Authors.
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

package verify

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore/pkg/signature"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tlog"
)

const maxAllowedTlogEntries = 32

// VerifyArtifactTransparencyLog verifies that the given entity has been logged
// in the transparency log and that the log entry is valid.
//
// The threshold parameter is the number of unique transparency log entries
// that must be verified.
func VerifyArtifactTransparencyLog(entity SignedEntity, trustedMaterial root.TrustedMaterial, logThreshold int, trustIntegratedTime bool) ([]root.Timestamp, error) { //nolint:revive
	entries, err := entity.TlogEntries()
	if err != nil {
		return nil, err
	}

	// limit the number of tlog entries to prevent DoS
	if len(entries) > maxAllowedTlogEntries {
		return nil, fmt.Errorf("too many tlog entries: %d > %d", len(entries), maxAllowedTlogEntries)
	}

	// disallow duplicate entries, as a malicious actor could use duplicates to bypass the threshold
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LogKeyID() == entries[j].LogKeyID() && entries[i].LogIndex() == entries[j].LogIndex() {
				return nil, errors.New("duplicate tlog entries found")
			}
		}
	}

	sigContent, err := entity.SignatureContent()
	if err != nil {
		return nil, err
	}

	entitySignature := sigContent.Signature()

	verificationContent, err := entity.VerificationContent()
	if err != nil {
		return nil, err
	}

	verifiedTimestamps := []root.Timestamp{}
	logEntriesVerified := 0

	for _, entry := range entries {
		err := tlog.ValidateEntry(entry)
		if err != nil {
			return nil, err
		}

		rekorLogs := trustedMaterial.RekorLogs()
		keyID := entry.LogKeyID()
		hex64Key := hex.EncodeToString([]byte(keyID))
		tlogVerifier, ok := trustedMaterial.RekorLogs()[hex64Key]
		if !ok {
			// skip entries the trust root cannot verify
			continue
		}

		if !entry.HasInclusionPromise() && !entry.HasInclusionProof() {
			return nil, fmt.Errorf("entry must contain an inclusion proof and/or promise")
		}
		if entry.HasInclusionPromise() {
			err = tlog.VerifySET(entry, rekorLogs)
			if err != nil {
				// skip entries the trust root cannot verify
				continue
			}
			if trustIntegratedTime {
				verifiedTimestamps = append(verifiedTimestamps, root.Timestamp{Time: entry.IntegratedTime(), URI: tlogVerifier.BaseURL})
			}
		}
		if entry.HasInclusionProof() {
			verifier, err := getVerifier(tlogVerifier.PublicKey, tlogVerifier.SignatureHashFunc)
			if err != nil {
				return nil, err
			}

			err = tlog.VerifyInclusion(entry, *verifier)
			if err != nil {
				return nil, err
			}
			// DO NOT use timestamp with only an inclusion proof, because it is not signed metadata
		}

		// Ensure entry signature matches signature from bundle
		if !bytes.Equal(entry.Signature(), entitySignature) {
			return nil, errors.New("transparency log signature does not match")
		}

		// Ensure entry certificate matches bundle certificate
		if !verificationContent.CompareKey(entry.PublicKey(), trustedMaterial) {
			return nil, errors.New("transparency log certificate does not match")
		}

		// TODO: if you have access to artifact, check that it matches body subject

		// Check tlog entry time against bundle certificates
		if !verificationContent.ValidAtTime(entry.IntegratedTime(), trustedMaterial) {
			return nil, errors.New("integrated time outside certificate validity")
		}

		// successful log entry verification
		logEntriesVerified++
	}

	if logEntriesVerified < logThreshold {
		return nil, fmt.Errorf("not enough verified log entries from transparency log: %d < %d", logEntriesVerified, logThreshold)
	}

	return verifiedTimestamps, nil
}

func getVerifier(publicKey crypto.PublicKey, hashFunc crypto.Hash) (*signature.Verifier, error) {
	verifier, err := signature.LoadVerifier(publicKey, hashFunc)
	if err != nil {
		return nil, err
	}

	return &verifier, nil
}
