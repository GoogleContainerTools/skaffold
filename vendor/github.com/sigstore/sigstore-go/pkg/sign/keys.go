// Copyright 2024 The Sigstore Authors.
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

package sign

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	_ "crypto/sha512" // if user chooses SHA2-384 or SHA2-512 for hash
	"crypto/x509"
	"encoding/base64"
	"fmt"

	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
)

type Keypair interface {
	GetHashAlgorithm() protocommon.HashAlgorithm
	GetSigningAlgorithm() protocommon.PublicKeyDetails
	GetHint() []byte
	GetKeyAlgorithm() string
	GetPublicKey() crypto.PublicKey
	GetPublicKeyPem() (string, error)
	SignData(ctx context.Context, data []byte) ([]byte, []byte, error)
}

type EphemeralKeypairOptions struct {
	// Optional fingerprint for public key
	Hint []byte
	// Optional algorithm for generating signing key
	Algorithm protocommon.PublicKeyDetails
}

type EphemeralKeypair struct {
	options    *EphemeralKeypairOptions
	privKey    crypto.Signer
	algDetails signature.AlgorithmDetails
}

// NewEphemeralKeypair generates a signing key to be used for a single signature generation.
// Defaults to ECDSA P-256 SHA-256 with a SHA-256 key hint.
func NewEphemeralKeypair(opts *EphemeralKeypairOptions) (*EphemeralKeypair, error) {
	if opts == nil {
		opts = &EphemeralKeypairOptions{}
	}

	// Default signing algorithm is ECDSA P-256 SHA-256
	if opts.Algorithm == protocommon.PublicKeyDetails_PUBLIC_KEY_DETAILS_UNSPECIFIED {
		opts.Algorithm = protocommon.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256
	}
	algDetails, err := signature.GetAlgorithmDetails(opts.Algorithm)
	if err != nil {
		return nil, err
	}
	var privKey crypto.Signer
	switch kt := algDetails.GetKeyType(); kt {
	case signature.ECDSA:
		curve, err := algDetails.GetECDSACurve()
		if err != nil {
			return nil, err
		}
		privKey, err = ecdsa.GenerateKey(*curve, rand.Reader)
		if err != nil {
			return nil, err
		}
	case signature.RSA:
		bitSize, err := algDetails.GetRSAKeySize()
		if err != nil {
			return nil, err
		}
		privKey, err = rsa.GenerateKey(rand.Reader, int(bitSize))
		if err != nil {
			return nil, err
		}
	case signature.ED25519:
		_, privKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %T", kt)
	}

	if opts.Hint == nil {
		pubKeyBytes, err := x509.MarshalPKIXPublicKey(privKey.Public())
		if err != nil {
			return nil, err
		}
		hashedBytes := sha256.Sum256(pubKeyBytes)
		opts.Hint = []byte(base64.StdEncoding.EncodeToString(hashedBytes[:]))
	}

	ephemeralKeypair := EphemeralKeypair{
		options:    opts,
		privKey:    privKey,
		algDetails: algDetails,
	}

	return &ephemeralKeypair, nil
}

// GetHashAlgorithm returns the hash algorithm to compute the digest to sign.
func (e *EphemeralKeypair) GetHashAlgorithm() protocommon.HashAlgorithm {
	return e.algDetails.GetProtoHashType()
}

// GetSigningAlgorithm returns the signing algorithm of the key.
func (e *EphemeralKeypair) GetSigningAlgorithm() protocommon.PublicKeyDetails {
	return e.algDetails.GetSignatureAlgorithm()
}

// GetHint returns the fingerprint of the public key.
func (e *EphemeralKeypair) GetHint() []byte {
	return e.options.Hint
}

// GetKeyAlgorithm returns the top-level key algorithm, used as part of requests
// to Fulcio. Prefer PublicKeyDetails for a more precise algorithm.
func (e *EphemeralKeypair) GetKeyAlgorithm() string {
	switch e.algDetails.GetKeyType() {
	case signature.ECDSA:
		return "ECDSA"
	case signature.RSA:
		return "RSA"
	case signature.ED25519:
		return "ED25519"
	default:
		return ""
	}
}

// GetPublicKey returns the public key.
func (e *EphemeralKeypair) GetPublicKey() crypto.PublicKey {
	return e.privKey.Public()
}

// GetPublicKeyPem returns the public key in PEM format.
func (e *EphemeralKeypair) GetPublicKeyPem() (string, error) {
	pubKeyBytes, err := cryptoutils.MarshalPublicKeyToPEM(e.privKey.Public())
	if err != nil {
		return "", err
	}

	return string(pubKeyBytes), nil
}

// SignData returns the signature and the data to sign, which is a digest except when
// signing with Ed25519.
func (e *EphemeralKeypair) SignData(_ context.Context, data []byte) ([]byte, []byte, error) {
	hf := e.algDetails.GetHashType()
	dataToSign := data
	// RSA, ECDSA, and Ed25519ph sign a digest, while pure Ed25519's interface takes data and hashes during signing
	if hf != crypto.Hash(0) {
		hasher := hf.New()
		hasher.Write(data)
		dataToSign = hasher.Sum(nil)
	}
	signature, err := e.privKey.Sign(rand.Reader, dataToSign, hf)
	if err != nil {
		return nil, nil, err
	}
	return signature, dataToSign, nil
}
