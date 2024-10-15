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

package ssh

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/sigstore/rekor/pkg/pki/identity"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"golang.org/x/crypto/ssh"
)

type Signature struct {
	signature *ssh.Signature
	pk        ssh.PublicKey
	hashAlg   string
}

// NewSignature creates and Validates an ssh signature object
func NewSignature(r io.Reader) (*Signature, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	sig, err := Decode(b)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// CanonicalValue implements the pki.Signature interface
func (s Signature) CanonicalValue() ([]byte, error) {
	return []byte(Armor(s.signature, s.pk)), nil
}

// Verify implements the pki.Signature interface
func (s Signature) Verify(r io.Reader, k interface{}, _ ...sigsig.VerifyOption) error {
	if s.signature == nil {
		return errors.New("ssh signature has not been initialized")
	}

	key, ok := k.(*PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key type for: %v", k)
	}

	ck, err := key.CanonicalValue()
	if err != nil {
		return err
	}
	cs, err := s.CanonicalValue()
	if err != nil {
		return err
	}
	return Verify(r, cs, ck)
}

// PublicKey contains an ssh PublicKey
type PublicKey struct {
	key     ssh.PublicKey
	comment string
}

// NewPublicKey implements the pki.PublicKey interface
func NewPublicKey(r io.Reader) (*PublicKey, error) {
	// 64K seems generous as a limit for valid SSH keys
	// we use http.MaxBytesReader and pass nil for ResponseWriter to reuse stdlib
	// and not reimplement this; There is a proposal for this to be fixed in 1.20
	// https://github.com/golang/go/issues/51115
	// TODO: switch this to stdlib once golang 1.20 comes out
	rawPub, err := io.ReadAll(http.MaxBytesReader(nil, io.NopCloser(r), 65536))
	if err != nil {
		return nil, err
	}

	key, comment, _, _, err := ssh.ParseAuthorizedKey(rawPub)
	if err != nil {
		return nil, err
	}

	return &PublicKey{key: key, comment: comment}, nil
}

// CanonicalValue implements the pki.PublicKey interface
func (k PublicKey) CanonicalValue() ([]byte, error) {
	if k.key == nil {
		return nil, errors.New("ssh public key has not been initialized")
	}
	return ssh.MarshalAuthorizedKey(k.key), nil
}

// EmailAddresses implements the pki.PublicKey interface
func (k PublicKey) EmailAddresses() []string {
	if govalidator.IsEmail(k.comment) {
		return []string{k.comment}
	}
	return nil
}

// Subjects implements the pki.PublicKey interface
func (k PublicKey) Subjects() []string {
	return k.EmailAddresses()
}

// Identities implements the pki.PublicKey interface
func (k PublicKey) Identities() ([]identity.Identity, error) {
	// extract key from SSH certificate if present
	var sshKey ssh.PublicKey
	switch v := k.key.(type) {
	case *ssh.Certificate:
		sshKey = v.Key
	default:
		sshKey = k.key
	}

	// Extract crypto.PublicKey from SSH key
	// Handle sk public keys which do not implement ssh.CryptoPublicKey
	// Inspired by x/ssh/keys.go
	// TODO: Simplify after https://github.com/golang/go/issues/62518
	var cryptoPubKey crypto.PublicKey
	if v, ok := sshKey.(ssh.CryptoPublicKey); ok {
		cryptoPubKey = v.CryptoPublicKey()
	} else {
		switch sshKey.Type() {
		case ssh.KeyAlgoSKECDSA256:
			var w struct {
				Curve       string
				KeyBytes    []byte
				Application string
				Rest        []byte `ssh:"rest"`
			}
			_, k, ok := parseString(sshKey.Marshal())
			if !ok {
				return nil, fmt.Errorf("error parsing ssh.KeyAlgoSKED25519 key")
			}
			if err := ssh.Unmarshal(k, &w); err != nil {
				return nil, err
			}
			if w.Curve != "nistp256" {
				return nil, errors.New("ssh: unsupported curve")
			}
			ecdsaPubKey := new(ecdsa.PublicKey)
			ecdsaPubKey.Curve = elliptic.P256()
			//nolint:staticcheck // ignore SA1019 for old code
			ecdsaPubKey.X, ecdsaPubKey.Y = elliptic.Unmarshal(ecdsaPubKey.Curve, w.KeyBytes)
			if ecdsaPubKey.X == nil || ecdsaPubKey.Y == nil {
				return nil, errors.New("ssh: invalid curve point")
			}
			cryptoPubKey = ecdsaPubKey
		case ssh.KeyAlgoSKED25519:
			var w struct {
				KeyBytes    []byte
				Application string
				Rest        []byte `ssh:"rest"`
			}
			_, k, ok := parseString(sshKey.Marshal())
			if !ok {
				return nil, fmt.Errorf("error parsing ssh.KeyAlgoSKED25519 key")
			}
			if err := ssh.Unmarshal(k, &w); err != nil {
				return nil, err
			}
			if l := len(w.KeyBytes); l != ed25519.PublicKeySize {
				return nil, fmt.Errorf("invalid size %d for Ed25519 public key", l)
			}
			cryptoPubKey = ed25519.PublicKey(w.KeyBytes)
		default:
			// Should not occur, as rsa, dsa, ecdsa, and ed25519 all implement ssh.CryptoPublicKey
			return nil, fmt.Errorf("unknown key type: %T", sshKey)
		}
	}

	pkixKey, err := cryptoutils.MarshalPublicKeyToDER(cryptoPubKey)
	if err != nil {
		return nil, err
	}
	fp := ssh.FingerprintSHA256(k.key)
	return []identity.Identity{{
		Crypto:      k.key,
		Raw:         pkixKey,
		Fingerprint: fp,
	}}, nil
}

// Copied by x/ssh/keys.go
// TODO: Remove after https://github.com/golang/go/issues/62518
func parseString(in []byte) (out, rest []byte, ok bool) {
	if len(in) < 4 {
		return
	}
	length := binary.BigEndian.Uint32(in)
	in = in[4:]
	if uint32(len(in)) < length {
		return
	}
	out = in[:length]
	rest = in[length:]
	ok = true
	return
}
