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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/sigstore-go/pkg/util"
	"github.com/sigstore/sigstore/pkg/oauthflow"
)

type CertificateProviderOptions struct {
	// Optional OIDC JWT to send to certificate provider; required for Fulcio
	IDToken string
}

type CertificateProvider interface {
	GetCertificate(context.Context, Keypair, *CertificateProviderOptions) ([]byte, error)
}

type Fulcio struct {
	options *FulcioOptions
	client  *http.Client
}

type FulcioOptions struct {
	// URL of Fulcio instance
	BaseURL string
	// Optional timeout for network requests (default 30s; use negative value for no timeout)
	Timeout time.Duration
	// Optional number of times to retry on HTTP 5XX
	Retries uint
	// Optional Transport (for dependency injection)
	Transport http.RoundTripper
}

var FulcioAPIVersions = []uint32{1}

type fulcioCertRequest struct {
	PublicKeyRequest publicKeyRequest `json:"publicKeyRequest"`
}

type publicKeyRequest struct {
	PublicKey         publicKey `json:"publicKey"`
	ProofOfPossession string    `json:"proofOfPossession"`
}

type publicKey struct {
	Algorithm string `json:"algorithm"`
	Content   string `json:"content"`
}

type fulcioResponse struct {
	SignedCertificateEmbeddedSct signedCertificateEmbeddedSct `json:"signedCertificateEmbeddedSct"`
	SignedCertificateDetachedSct signedCertificateDetachedSct `json:"signedCertificateDetachedSct"`
}

type signedCertificateEmbeddedSct struct {
	Chain chain `json:"chain"`
}

type signedCertificateDetachedSct struct {
	Chain chain `json:"chain"`
}

type chain struct {
	Certificates []string `json:"certificates"`
}

func NewFulcio(opts *FulcioOptions) *Fulcio {
	fulcio := &Fulcio{options: opts}
	fulcio.client = &http.Client{
		Transport: opts.Transport,
	}

	if opts.Timeout >= 0 {
		if opts.Timeout == 0 {
			opts.Timeout = 30 * time.Second
		}
		fulcio.client.Timeout = opts.Timeout
	}

	return fulcio
}

// Returns DER-encoded code signing certificate
func (f *Fulcio) GetCertificate(ctx context.Context, keypair Keypair, opts *CertificateProviderOptions) ([]byte, error) {
	if opts.IDToken == "" {
		return nil, errors.New("fetching certificate from Fulcio requires IDToken to be set")
	}

	// Get JWT from identity token
	//
	// Note that the contents of this token are untrusted. Fulcio will perform
	// the token verification.
	tokenParts := strings.Split(opts.IDToken, ".")
	if len(tokenParts) < 2 {
		return nil, errors.New("unable to get subject from identity token")
	}

	jwtString, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return nil, err
	}

	subject, err := oauthflow.SubjectFromUnverifiedToken(jwtString)
	if err != nil {
		return nil, err
	}

	// Fulcio doesn't support verifying Ed25519ph signatures currently.
	if keypair.GetSigningAlgorithm() == protocommon.PublicKeyDetails_PKIX_ED25519_PH {
		return nil, fmt.Errorf("ed25519ph unsupported by Fulcio")
	}

	// Sign JWT subject for proof of possession
	subjectSignature, _, err := keypair.SignData(ctx, []byte(subject))
	if err != nil {
		return nil, err
	}

	// Make Fulcio certificate request
	keypairPem, err := keypair.GetPublicKeyPem()
	if err != nil {
		return nil, err
	}

	certRequest := fulcioCertRequest{
		PublicKeyRequest: publicKeyRequest{
			PublicKey: publicKey{
				Algorithm: keypair.GetKeyAlgorithm(),
				Content:   keypairPem,
			},
			ProofOfPossession: base64.StdEncoding.EncodeToString(subjectSignature),
		},
	}

	requestJSON, err := json.Marshal(&certRequest)
	if err != nil {
		return nil, err
	}

	// TODO: For now we are using our own HTTP client
	//
	// https://github.com/sigstore/fulcio/pkg/api's client could be used in the
	// future, when it supports the v2 API
	attempts := uint(0)
	var response *http.Response

	for attempts <= f.options.Retries {
		request, err := http.NewRequest("POST", f.options.BaseURL+"/api/v2/signingCert", bytes.NewBuffer(requestJSON))
		if err != nil {
			return nil, err
		}
		request.Header.Add("Authorization", "Bearer "+opts.IDToken)
		request.Header.Add("Content-Type", "application/json")
		request.Header.Add("User-Agent", util.ConstructUserAgent())

		response, err = f.client.Do(request)
		if err != nil {
			return nil, err
		}

		if (response.StatusCode < 500 || response.StatusCode >= 600) && response.StatusCode != 429 {
			// Not a retryable HTTP status code, so don't retry
			break
		}

		delay := time.Duration(math.Pow(2, float64(attempts)))
		timer := time.NewTimer(delay * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		attempts++
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Fulcio returned %d: %s", response.StatusCode, string(body))
	}

	// Assemble bundle from Fulcio response
	var fulcioResp fulcioResponse
	err = json.Unmarshal(body, &fulcioResp)
	if err != nil {
		return nil, err
	}

	var cert []byte
	switch {
	case len(fulcioResp.SignedCertificateEmbeddedSct.Chain.Certificates) > 0:
		cert = []byte(fulcioResp.SignedCertificateEmbeddedSct.Chain.Certificates[0])
	case len(fulcioResp.SignedCertificateDetachedSct.Chain.Certificates) > 0:
		cert = []byte(fulcioResp.SignedCertificateDetachedSct.Chain.Certificates[0])
	default:
		return nil, errors.New("Fulcio returned no certificates")
	}

	certBlock, _ := pem.Decode(cert)
	if certBlock == nil {
		return nil, errors.New("unable to parse Fulcio certificate")
	}

	return certBlock.Bytes, nil
}
