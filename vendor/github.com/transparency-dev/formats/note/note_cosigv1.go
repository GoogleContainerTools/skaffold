// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package note

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/mod/sumdb/note"
)

const (
	algEd25519              = 1
	algECDSAWithSHA256      = 2
	algEd25519CosignatureV1 = 4
	algRFC6962STH           = 5
)

const (
	keyHashSize   = 4
	timestampSize = 8
)

// NewSignerForCosignatureV1 constructs a new Signer that produces timestamped
// cosignature/v1 signatures from a standard Ed25519 encoded signer key.
//
// (The returned Signer has a different key hash from a non-timestamped one,
// meaning it will differ from the key hash in the input encoding.)
func NewSignerForCosignatureV1(skey string) (*Signer, error) {
	priv1, skey, _ := strings.Cut(skey, "+")
	priv2, skey, _ := strings.Cut(skey, "+")
	name, skey, _ := strings.Cut(skey, "+")
	hash16, key64, _ := strings.Cut(skey, "+")
	key, err := base64.StdEncoding.DecodeString(key64)
	if priv1 != "PRIVATE" || priv2 != "KEY" || len(hash16) != 8 || err != nil || !isValidName(name) || len(key) == 0 {
		return nil, errSignerID
	}

	s := &Signer{name: name}

	alg, key := key[0], key[1:]
	switch alg {
	default:
		return nil, errSignerAlg

	case algEd25519:
		if len(key) != ed25519.SeedSize {
			return nil, errSignerID
		}
		key := ed25519.NewKeyFromSeed(key)
		pubkey := append([]byte{algEd25519CosignatureV1}, key.Public().(ed25519.PublicKey)...)
		s.hash = keyHashEd25519(name, pubkey)
		s.sign = func(msg []byte) ([]byte, error) {
			t := uint64(time.Now().Unix())
			m, err := formatCosignatureV1(t, msg)
			if err != nil {
				return nil, err
			}

			// The signature itself is encoded as timestamp || signature.
			sig := make([]byte, 0, timestampSize+ed25519.SignatureSize)
			sig = binary.BigEndian.AppendUint64(sig, t)
			sig = append(sig, ed25519.Sign(key, m)...)
			return sig, nil
		}
		s.verify = verifyCosigV1(pubkey[1:])
	}

	return s, nil
}

// NewVerifierForCosignatureV1 constructs a new Verifier for timestamped
// cosignature/v1 signatures from a standard Ed25519 encoded verifier key.
//
// (The returned Verifier has a different key hash from a non-timestamped one,
// meaning it will differ from the key hash in the input encoding.)
func NewVerifierForCosignatureV1(vkey string) (note.Verifier, error) {
	name, vkey, _ := strings.Cut(vkey, "+")
	hash16, key64, _ := strings.Cut(vkey, "+")
	key, err := base64.StdEncoding.DecodeString(key64)
	if len(hash16) != 8 || err != nil || !isValidName(name) || len(key) == 0 {
		return nil, errVerifierID
	}

	v := &verifier{
		name: name,
	}

	alg, key := key[0], key[1:]
	switch alg {
	default:
		return nil, errVerifierAlg

	case algEd25519:
		if len(key) != 32 {
			return nil, errVerifierID
		}
		v.keyHash = keyHashEd25519(name, append([]byte{algEd25519CosignatureV1}, key...))
		v.v = verifyCosigV1(key)
	}

	return v, nil
}

// CoSigV1Timestamp extracts the embedded timestamp from a CoSigV1 signature.
func CoSigV1Timestamp(s note.Signature) (time.Time, error) {
	r, err := base64.StdEncoding.DecodeString(s.Base64)
	if err != nil {
		return time.UnixMilli(0), errMalformedSig
	}
	if len(r) != keyHashSize+timestampSize+ed25519.SignatureSize {
		return time.UnixMilli(0), errVerifierAlg
	}
	r = r[keyHashSize:] // Skip the hash
	// Next 8 bytes are the timestamp as Unix seconds-since-epoch:
	return time.Unix(int64(binary.BigEndian.Uint64(r)), 0), nil
}

// verifyCosigV1 returns a verify function based on key.
func verifyCosigV1(key []byte) func(msg, sig []byte) bool {
	return func(msg, sig []byte) bool {
		if len(sig) != timestampSize+ed25519.SignatureSize {
			return false
		}
		t := binary.BigEndian.Uint64(sig)
		sig = sig[timestampSize:]
		m, err := formatCosignatureV1(t, msg)
		if err != nil {
			return false
		}
		return ed25519.Verify(key, m, sig)
	}
}

func formatCosignatureV1(t uint64, msg []byte) ([]byte, error) {
	// The signed message is in the following format
	//
	//      cosignature/v1
	//      time TTTTTTTTTT
	//      origin line
	//      NNNNNNNNN
	//      tree hash
	//      ...
	//
	// where TTTTTTTTTT is the current UNIX timestamp, and the following
	// lines are the lines of the note.
	//
	// While the witness signs all lines of the note, it's important to
	// understand that the witness is asserting observation of correct
	// append-only operation of the log based on the first three lines;
	// no semantic statement is made about any extra "extension" lines.

	if lines := bytes.Split(msg, []byte("\n")); len(lines) < 3 {
		return nil, errors.New("cosigned note format invalid")
	}
	return []byte(fmt.Sprintf("cosignature/v1\ntime %d\n%s", t, msg)), nil
}

var (
	errSignerID     = errors.New("malformed signer id")
	errSignerAlg    = errors.New("unknown signer algorithm")
	errVerifierID   = errors.New("malformed verifier id")
	errVerifierAlg  = errors.New("unknown verifier algorithm")
	errMalformedSig = errors.New("malformed signature")
)

type Signer struct {
	name   string
	hash   uint32
	sign   func([]byte) ([]byte, error)
	verify func(msg, sig []byte) bool
}

func (s *Signer) Name() string                    { return s.name }
func (s *Signer) KeyHash() uint32                 { return s.hash }
func (s *Signer) Sign(msg []byte) ([]byte, error) { return s.sign(msg) }

func (s *Signer) Verifier() note.Verifier {
	return &verifier{
		name:    s.name,
		keyHash: s.hash,
		v:       s.verify,
	}
}

// isValidName reports whether name is valid.
// It must be non-empty and not have any Unicode spaces or pluses.
func isValidName(name string) bool {
	return name != "" && utf8.ValidString(name) && strings.IndexFunc(name, unicode.IsSpace) < 0 && !strings.Contains(name, "+")
}

func keyHashEd25519(name string, key []byte) uint32 {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte("\n"))
	h.Write(key)
	sum := h.Sum(nil)
	return binary.BigEndian.Uint32(sum)
}
