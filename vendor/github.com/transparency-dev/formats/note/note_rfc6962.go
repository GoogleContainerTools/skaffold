// Copyright 2024 Google LLC. All Rights Reserved.
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

package note

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/sumdb/note"
)

// RFC6962VerifierString creates a note style verifier string for use with NewRFC6962Verifier below.
// logURL is the root URL of the log.
// pubK is the public key of the log.
func RFC6962VerifierString(logURL string, pubK crypto.PublicKey) (string, error) {
	if !isValidName(logURL) {
		return "", errors.New("invalid name")
	}
	pubSer, err := x509.MarshalPKIXPublicKey(pubK)
	if err != nil {
		return "", err
	}
	logID := sha256.Sum256(pubSer)
	name := rfc6962LogName(logURL)
	hash := rfc6962Keyhash(name, logID)
	return fmt.Sprintf("%s+%08x+%s", name, hash, base64.StdEncoding.EncodeToString(append([]byte{algRFC6962STH}, pubSer...))), nil
}

// NewRFC6962Verifier creates a note verifier for Sunlight/RFC6962 checkpoint signatures.
func NewRFC6962Verifier(vkey string) (note.Verifier, error) {
	name, vkey, _ := strings.Cut(vkey, "+")
	hash16, key64, _ := strings.Cut(vkey, "+")
	key, err := base64.StdEncoding.DecodeString(key64)
	if len(hash16) != 8 || err != nil || !isValidName(name) || len(key) == 0 {
		return nil, errVerifierID
	}

	v := &rfc6962Verifer{
		name: name,
	}

	alg, key := key[0], key[1:]
	if alg != algRFC6962STH {
		return nil, errVerifierAlg
	}

	pubK, err := x509.ParsePKIXPublicKey(key)
	if err != nil {
		return nil, errors.New("invalid key")
	}

	logID := sha256.Sum256(key)
	v.keyHash = rfc6962Keyhash(name, logID)
	v.v = verifyRFC6962(pubK)

	return v, nil
}

// signedTreeHead represents the structure returned by the get-sth CT method
// after base64 decoding; see sections 3.5 and 4.3.
type signedTreeHead struct {
	Version           int    `json:"sth_version"`         // The version of the protocol to which the STH conforms
	TreeSize          uint64 `json:"tree_size"`           // The number of entries in the new tree
	Timestamp         uint64 `json:"timestamp"`           // The time at which the STH was created
	SHA256RootHash    []byte `json:"sha256_root_hash"`    // The root hash of the log's Merkle tree
	TreeHeadSignature []byte `json:"tree_head_signature"` // Log's signature over a TLS-encoded TreeHeadSignature
	LogID             []byte `json:"log_id"`              // The SHA256 hash of the log's public key
}

// RFC6962STHToCheckpoint converts the provided RFC6962 JSON representation of a CT Signed Tree Head structure to
// a sunlight style signed checkpoint.
// The passed in verifier must be an RFC6929Verifier containing the correct details for the log which signed the STH.
func RFC6962STHToCheckpoint(j []byte, v note.Verifier) ([]byte, error) {
	var sth signedTreeHead
	if err := json.Unmarshal(j, &sth); err != nil {
		return nil, err
	}
	logName := v.Name()
	body := fmt.Sprintf("%s\n%d\n%s\n", logName, sth.TreeSize, base64.StdEncoding.EncodeToString(sth.SHA256RootHash))
	sigBytes := binary.BigEndian.AppendUint32(nil, v.KeyHash())
	sigBytes = binary.BigEndian.AppendUint64(sigBytes, sth.Timestamp)
	sigBytes = append(sigBytes, sth.TreeHeadSignature...)
	sigLine := fmt.Sprintf("\u2014 %s %s", logName, base64.StdEncoding.EncodeToString(sigBytes))

	n := []byte(fmt.Sprintf("%s\n%s\n", body, sigLine))
	if _, err := note.Open(n, note.VerifierList(v)); err != nil {
		return nil, err
	}
	return n, nil
}

// RFC6962STHTimestamp extracts the embedded timestamp from a translated RFC6962 STH signature.
func RFC6962STHTimestamp(s note.Signature) (time.Time, error) {
	r, err := base64.StdEncoding.DecodeString(s.Base64)
	if err != nil {
		return time.UnixMilli(0), errMalformedSig
	}
	if len(r) <= keyHashSize+timestampSize {
		return time.UnixMilli(0), errVerifierAlg
	}
	r = r[keyHashSize:] // Skip the hash
	// Next 8 bytes are the timestamp as Unix millis-since-epoch:
	return time.Unix(0, int64(binary.BigEndian.Uint64(r)*1000)), nil
}

func rfc6962Keyhash(name string, logID [32]byte) uint32 {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte{0x0A, algRFC6962STH})
	h.Write(logID[:])
	r := h.Sum(nil)
	return binary.BigEndian.Uint32(r)
}

// rfc6962LogName returns a sunlight checkpoint compatible log name from the
// passed in CT log root URL.
//
// "For example, a log with submission prefix https://rome.ct.example.com/2024h1/ will use rome.ct.example.com/2024h1 as the checkpoint origin line"
// (see https://github.com/C2SP/C2SP/blob/main/sunlight.md#checkpoints)
func rfc6962LogName(logURL string) string {
	logURL = strings.ToLower(logURL)
	logURL = strings.TrimPrefix(logURL, "http://")
	logURL = strings.TrimPrefix(logURL, "https://")
	logURL = strings.TrimSuffix(logURL, "/")
	return logURL
}

type rfc6962Verifer struct {
	name    string
	keyHash uint32
	v       func(msg []byte, origin string, sig []byte) bool
}

// Name returns the name associated with the key this verifier is based on.
func (v *rfc6962Verifer) Name() string {
	return v.name
}

// KeyHash returns a truncated hash of the key this verifier is based on.
func (v *rfc6962Verifer) KeyHash() uint32 {
	return v.keyHash
}

// Verify checks that the provided sig is valid over msg for the key this verifier is based on.
func (v *rfc6962Verifer) Verify(msg, sig []byte) bool {
	return v.v(msg, v.name, sig)
}

func verifyRFC6962(key crypto.PublicKey) func([]byte, string, []byte) bool {
	return func(msg []byte, origin string, sig []byte) bool {
		if len(sig) < timestampSize {
			return false
		}
		t := binary.BigEndian.Uint64(sig)
		// slice off timestamp bytes
		sig = sig[timestampSize:]

		hAlg := sig[0]
		sAlg := sig[1]
		// slice off the hAlg and sAlg bytes read above
		sig = sig[2:]

		// Figure out sig bytes length
		sigLen := binary.BigEndian.Uint16(sig)
		sig = sig[2:] // Slice off length bytes

		// All that remains should be the signature bytes themselves, and nothing more.
		if len(sig) != int(sigLen) {
			return false
		}

		// SHA256 (RFC 5246 s7.4.1.4.1.)
		if hAlg != 0x04 {
			return false
		}

		o, m, err := formatRFC6962STH(t, msg)
		if err != nil {
			return false
		}
		if origin != o {
			return false
		}
		dgst := sha256.Sum256(m)
		switch k := key.(type) {
		case *ecdsa.PublicKey:
			// RFC 5246 s7.4.1.4.1.
			if sAlg != 0x03 {
				return false
			}
			return ecdsa.VerifyASN1(k, dgst[:], sig)
		case *rsa.PublicKey:
			// RFC 5246 s7.4.1.4.1.
			if sAlg != 0x01 {
				return false
			}
			return rsa.VerifyPKCS1v15(k, crypto.SHA256, dgst[:], sig) != nil
		default:
			return false
		}
	}
}

// formatRFC6962STH uses the provided timestamp and checkpoint body to
// recreate the RFC6962 STH structure over which the signature was made.
func formatRFC6962STH(t uint64, msg []byte) (string, []byte, error) {
	// Must be:
	// origin (schema-less log root url) "\n"
	// tree size (decimal) "\n"
	// root hash (b64) "\n"
	lines := strings.Split(string(msg), "\n")
	if len(lines) != 4 {
		return "", nil, errors.New("wrong number of lines")
	}
	if len(lines[3]) != 0 {
		return "", nil, errors.New("extension line(s) present")
	}
	size, err := strconv.ParseUint(lines[1], 10, 64)
	if err != nil {
		return "", nil, err
	}
	root, err := base64.StdEncoding.DecodeString(lines[2])
	if err != nil {
		return "", nil, err
	}
	if len(root) != 32 {
		return "", nil, errors.New("invalid root hash size")
	}
	rootHash := [32]byte{}
	copy(rootHash[:], root)

	sth := treeHeadSignature{
		Version:        V1,
		TreeSize:       size,
		Timestamp:      t,
		SHA256RootHash: rootHash,
	}
	input, err := sth.Marshal()
	if err != nil {
		return "", nil, err
	}
	return lines[0], input, nil
}

// CT Version constants from section 3.2.
const (
	V1 uint8 = 0
)

// SignatureType constants from section 3.2.
const (
	treeHashSignatureType uint8 = 1
)

// treeHeadSignature holds the data over which the signature in an STH is
// generated; see section 3.5
type treeHeadSignature struct {
	// Version represents the Version enum from section 3.2:
	//
	//	enum { v1(0), (255) } Version;
	Version uint8
	// SignatureType differentiates STH signatures from SCT signatures, see section 3.2.
	//
	//	enum { certificate_timestamp(0), tree_hash(1), (255) } SignatureType;
	SignatureType uint8
	Timestamp     uint64
	TreeSize      uint64
	// sha256Hash represents the output from the SHA256 hash function.
	SHA256RootHash [sha256.Size]byte
}

// Marshal serializes the passed in STH into the correct
// format for signing.
func (s treeHeadSignature) Marshal() ([]byte, error) {
	switch s.Version {
	case V1:
		if len(s.SHA256RootHash) != crypto.SHA256.Size() {
			return nil, fmt.Errorf("invalid TreeHash length, got %d expected %d", len(s.SHA256RootHash), crypto.SHA256.Size())
		}
		s.SignatureType = treeHashSignatureType

		// This is technically TLS encoded, but since all fields are of known size it boils down to
		// just the raw bytes.
		b := make([]byte, 2+8+8+32)
		i := 0
		b[i] = byte(s.Version)
		i++
		b[i] = byte(treeHashSignatureType)
		i++
		binary.BigEndian.PutUint64(b[i:], s.Timestamp)
		i += 8
		binary.BigEndian.PutUint64(b[i:], s.TreeSize)
		i += 8
		copy(b[i:], s.SHA256RootHash[:])

		return b, nil
	default:
		return nil, fmt.Errorf("unsupported STH version %d", s.Version)
	}
}
