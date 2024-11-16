//
// Copyright 2021 The Sigstore Authors.
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

package tuf

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/sigstore/rekor/pkg/pki/identity"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/pkg/keys"
	"github.com/theupdateframework/go-tuf/verify"
)

type Signature struct {
	signed  *data.Signed
	Role    string
	Version int
}

type signedMeta struct {
	Type        string    `json:"_type"`
	Expires     time.Time `json:"expires"`
	Version     int       `json:"version"`
	SpecVersion string    `json:"spec_version"`
}

// NewSignature creates and validates a TUF signed manifest
func NewSignature(r io.Reader) (*Signature, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	s := &data.Signed{}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, err
	}

	// extract role
	sm := &signedMeta{}
	if err := json.Unmarshal(s.Signed, sm); err != nil {
		return nil, err
	}

	return &Signature{
		signed:  s,
		Role:    sm.Type,
		Version: sm.Version,
	}, nil
}

// CanonicalValue implements the pki.Signature interface
func (s Signature) CanonicalValue() ([]byte, error) {
	if s.signed == nil {
		return nil, errors.New("tuf manifest has not been initialized")
	}
	marshalledBytes, err := json.Marshal(s.signed)
	if err != nil {
		return nil, fmt.Errorf("marshalling signature: %w", err)
	}
	return jsoncanonicalizer.Transform(marshalledBytes)
}

// Verify implements the pki.Signature interface
func (s Signature) Verify(_ io.Reader, k interface{}, _ ...sigsig.VerifyOption) error {
	key, ok := k.(*PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key type for: %v", k)
	}

	if key.db == nil {
		return errors.New("tuf root has not been initialized")
	}

	return key.db.Verify(s.signed, s.Role, 0)
}

// PublicKey Public Key database with verification keys
type PublicKey struct {
	// we keep the signed root to retrieve the canonical value
	root *data.Signed
	db   *verify.DB
}

// NewPublicKey implements the pki.PublicKey interface
func NewPublicKey(r io.Reader) (*PublicKey, error) {
	rawRoot, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Unmarshal this to verify that this is a valid root.json
	s := &data.Signed{}
	if err := json.Unmarshal(rawRoot, s); err != nil {
		return nil, err
	}
	root := &data.Root{}
	if err := json.Unmarshal(s.Signed, root); err != nil {
		return nil, err
	}

	// Now create a verification db that trusts all the keys
	db := verify.NewDB()
	for id, k := range root.Keys {
		if err := db.AddKey(id, k); err != nil {
			return nil, err
		}
	}
	for name, role := range root.Roles {
		if err := db.AddRole(name, role); err != nil {
			return nil, err
		}
	}

	// Verify that this root.json was signed.
	if err := db.Verify(s, "root", 0); err != nil {
		return nil, err
	}

	return &PublicKey{root: s, db: db}, nil
}

// CanonicalValue implements the pki.PublicKey interface
func (k PublicKey) CanonicalValue() (encoded []byte, err error) {
	if k.root == nil {
		return nil, errors.New("tuf root has not been initialized")
	}
	marshalledBytes, err := json.Marshal(k.root)
	if err != nil {
		return nil, fmt.Errorf("marshalling tuf root: %w", err)
	}
	return jsoncanonicalizer.Transform(marshalledBytes)
}

func (k PublicKey) SpecVersion() (string, error) {
	// extract role
	sm := &signedMeta{}
	if err := json.Unmarshal(k.root.Signed, sm); err != nil {
		return "", err
	}
	return sm.SpecVersion, nil
}

// EmailAddresses implements the pki.PublicKey interface
func (k PublicKey) EmailAddresses() []string {
	return nil
}

// Subjects implements the pki.PublicKey interface
func (k PublicKey) Subjects() []string {
	return nil
}

// Identities implements the pki.PublicKey interface
func (k PublicKey) Identities() ([]identity.Identity, error) {
	root := &data.Root{}
	if err := json.Unmarshal(k.root.Signed, root); err != nil {
		return nil, err
	}
	var ids []identity.Identity
	for _, k := range root.Keys {
		verifier, err := keys.GetVerifier(k)
		if err != nil {
			return nil, err
		}
		switch k.Type {
		// RSA and ECDSA keys are PKIX-encoded without PEM header for the Verifier type
		case data.KeyTypeRSASSA_PSS_SHA256:
			fallthrough
		// TODO: Update to constants once go-tuf is updated to 0.6.0 (need PR #508)
		case "ecdsa-sha2-nistp256":
			fallthrough
		case "ecdsa":
			// parse and marshal to check format is correct
			pub, err := x509.ParsePKIXPublicKey([]byte(verifier.Public()))
			if err != nil {
				return nil, err
			}
			pkixKey, err := cryptoutils.MarshalPublicKeyToDER(pub)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(pkixKey)
			ids = append(ids, identity.Identity{
				Crypto:      pub,
				Raw:         pkixKey,
				Fingerprint: hex.EncodeToString(digest[:]),
			})
		case data.KeyTypeEd25519:
			// key is stored as a 32-byte string
			pub := ed25519.PublicKey(verifier.Public())
			pkixKey, err := cryptoutils.MarshalPublicKeyToDER(pub)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(pkixKey)
			ids = append(ids, identity.Identity{
				Crypto:      pub,
				Raw:         pkixKey,
				Fingerprint: hex.EncodeToString(digest[:]),
			})
		default:
			return nil, fmt.Errorf("unsupported key type: %v", k.Type)
		}
	}
	return ids, nil
}
