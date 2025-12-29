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
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	rekortilesclient "github.com/sigstore/rekor-tiles/v2/pkg/client"
	"github.com/sigstore/rekor-tiles/v2/pkg/client/write"
	rekortilespb "github.com/sigstore/rekor-tiles/v2/pkg/generated/protobuf"
	"github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/pki"
	"github.com/sigstore/rekor/pkg/tle"
	"github.com/sigstore/rekor/pkg/types"
	"github.com/sigstore/rekor/pkg/types/dsse"
	"github.com/sigstore/rekor/pkg/types/hashedrekord"
	rekorUtil "github.com/sigstore/rekor/pkg/util"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"

	// To initialize rekor types
	_ "github.com/sigstore/rekor/pkg/types/dsse/v0.0.1"
	_ "github.com/sigstore/rekor/pkg/types/hashedrekord/v0.0.1"

	"github.com/sigstore/sigstore-go/pkg/util"
)

const (
	rekorV1 = iota + 1
	rekorV2
)

type RekorClient interface {
	CreateLogEntry(params *entries.CreateLogEntryParams, opts ...entries.ClientOption) (*entries.CreateLogEntryCreated, error)
}

type RekorV2Client interface {
	Add(ctx context.Context, entry any) (*protorekor.TransparencyLogEntry, error)
}

type Transparency interface {
	GetTransparencyLogEntry(context.Context, []byte, *protobundle.Bundle) error
}

type Rekor struct {
	options *RekorOptions
}

type RekorOptions struct {
	// URL of Fulcio instance
	BaseURL string
	// Optional timeout for network requests (default 30s; use negative value for no timeout)
	Timeout time.Duration
	// Optional number of times to retry
	Retries uint
	// Optional client (for dependency injection)
	Client   RekorClient
	ClientV2 RekorV2Client
	Version  uint32
}

var RekorAPIVersions = []uint32{1, 2}

func NewRekor(opts *RekorOptions) *Rekor {
	if opts.Version == 0 {
		opts.Version = rekorV1
	}
	return &Rekor{options: opts}
}

func (r *Rekor) GetTransparencyLogEntry(ctx context.Context, keyOrCertPEM []byte, b *protobundle.Bundle) error {
	var tlogEntry *protorekor.TransparencyLogEntry
	switch r.options.Version {
	case rekorV1:
		var err error
		tlogEntry, err = r.getRekorV1TLE(ctx, keyOrCertPEM, b)
		if err != nil {
			return err
		}
	case rekorV2:
		var err error
		tlogEntry, err = r.getRekorV2TLE(ctx, keyOrCertPEM, b)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown rekor version: %d", r.options.Version)
	}

	if b.VerificationMaterial.TlogEntries == nil {
		b.VerificationMaterial.TlogEntries = []*protorekor.TransparencyLogEntry{}
	}

	b.VerificationMaterial.TlogEntries = append(b.VerificationMaterial.TlogEntries, tlogEntry)

	return nil
}

func (r *Rekor) getRekorV2TLE(ctx context.Context, keyOrCertPEM []byte, b *protobundle.Bundle) (*protorekor.TransparencyLogEntry, error) {
	dsseEnvelope := b.GetDsseEnvelope()
	messageSignature := b.GetMessageSignature()
	verificationMaterial := b.GetVerificationMaterial()
	bundleCertificate := verificationMaterial.GetCertificate()

	block, _ := pem.Decode(keyOrCertPEM)
	keyOrCertDER := block.Bytes

	// Determine the signing algorithm for the public key
	var pubKey crypto.PublicKey
	var err error
	switch block.Type {
	case "PUBLIC KEY":
		pubKey, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	case "CERTIFICATE":
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		pubKey = c.PublicKey
	default:
		return nil, fmt.Errorf("unknown key type: %s", block.Type)
	}
	var opts []signature.LoadOption
	// When signing with ed25519, only the prehash variant is supported for hashedrekord
	if messageSignature != nil {
		opts = append(opts, options.WithED25519ph())
	}
	algoDetails, err := signature.GetDefaultAlgorithmDetails(pubKey, opts...)
	if err != nil {
		return nil, fmt.Errorf("getting algorithm details: %w", err)
	}

	verifier := &rekortilespb.Verifier{KeyDetails: algoDetails.GetSignatureAlgorithm()}
	if bundleCertificate != nil {
		verifier.Verifier = &rekortilespb.Verifier_X509Certificate{
			X509Certificate: &protocommon.X509Certificate{
				RawBytes: keyOrCertDER,
			},
		}
	} else {
		verifier.Verifier = &rekortilespb.Verifier_PublicKey{
			PublicKey: &rekortilespb.PublicKey{
				RawBytes: keyOrCertDER,
			},
		}
	}

	var req any
	switch {
	case dsseEnvelope != nil:
		req = &rekortilespb.DSSERequestV002{
			Envelope:  dsseEnvelope,
			Verifiers: []*rekortilespb.Verifier{verifier},
		}
	case messageSignature != nil:
		req = &rekortilespb.HashedRekordRequestV002{
			Signature: &rekortilespb.Signature{
				Content:  messageSignature.Signature,
				Verifier: verifier,
			},
			Digest: messageSignature.MessageDigest.Digest,
		}
	default:
		return nil, fmt.Errorf("unable to find signature in bundle")
	}
	if r.options.ClientV2 == nil {
		client, err := write.NewWriter(r.options.BaseURL, rekortilesclient.WithUserAgent(util.ConstructUserAgent()))
		if err != nil {
			return nil, fmt.Errorf("creating rekor v2 client: %w", err)
		}
		r.options.ClientV2 = client
	}
	tle, err := r.options.ClientV2.Add(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("adding rekor v2 entry: %w", err)
	}
	return tle, nil
}

func (r *Rekor) getRekorV1TLE(ctx context.Context, keyOrCertPEM []byte, b *protobundle.Bundle) (*protorekor.TransparencyLogEntry, error) {
	dsseEnvelope := b.GetDsseEnvelope()
	messageSignature := b.GetMessageSignature()

	artifactProperties := types.ArtifactProperties{
		PublicKeyBytes: [][]byte{keyOrCertPEM},
	}

	var proposedEntry models.ProposedEntry

	switch {
	case dsseEnvelope != nil:
		dsseType := dsse.New()

		artifactBytes, err := json.Marshal(dsseEnvelope)
		if err != nil {
			return nil, err
		}

		artifactProperties.ArtifactBytes = artifactBytes

		proposedEntry, err = dsseType.CreateProposedEntry(ctx, "", artifactProperties)
		if err != nil {
			return nil, err
		}
	case messageSignature != nil:
		hashedrekordType := hashedrekord.New()

		hexDigest := hex.EncodeToString(messageSignature.MessageDigest.Digest)

		artifactProperties.PKIFormat = string(pki.X509)
		artifactProperties.SignatureBytes = messageSignature.Signature
		artifactProperties.ArtifactHash = rekorUtil.PrefixSHA(hexDigest)

		var err error
		proposedEntry, err = hashedrekordType.CreateProposedEntry(ctx, "", artifactProperties)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unable to find signature in bundle")
	}

	params := entries.NewCreateLogEntryParams()
	if r.options.Timeout >= 0 {
		if r.options.Timeout == 0 {
			r.options.Timeout = 30 * time.Second
		}
		params.SetTimeout(r.options.Timeout)
	}
	params.SetProposedEntry(proposedEntry)
	params.SetContext(ctx)

	if r.options.Client == nil {
		client, err := client.GetRekorClient(r.options.BaseURL, client.WithUserAgent(util.ConstructUserAgent()), client.WithRetryCount(r.options.Retries))
		if err != nil {
			return nil, err
		}
		r.options.Client = client.Entries
	}

	resp, err := r.options.Client.CreateLogEntry(params)
	if err != nil {
		return nil, err
	}

	entry := resp.Payload[resp.ETag]
	tlogEntry, err := tle.GenerateTransparencyLogEntry(entry)
	if err != nil {
		return nil, err
	}
	return tlogEntry, nil
}
