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

package bundle

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sigstore/cosign/v3/internal/ui"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore/pkg/signature"
	"google.golang.org/protobuf/encoding/protojson"
)

type SignOptions struct {
	TSAClientTransport  http.RoundTripper
	CertificateProvider sign.CertificateProvider
}

func SignData(ctx context.Context, content sign.Content, keypair sign.Keypair, idToken string, cert []byte, signingConfig *root.SigningConfig, trustedMaterial root.TrustedMaterial, opts SignOptions) ([]byte, error) {
	var bundleOpts sign.BundleOptions

	if trustedMaterial != nil {
		bundleOpts.TrustedRoot = trustedMaterial
	}

	switch {
	case opts.CertificateProvider != nil:
		bundleOpts.CertificateProvider = opts.CertificateProvider
		if idToken != "" {
			bundleOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
				IDToken: idToken,
			}
		}
	case idToken != "":
		provider, err := newFulcioProvider(signingConfig)
		if err != nil {
			return nil, err
		}
		bundleOpts.CertificateProvider = provider
		bundleOpts.CertificateProviderOptions = &sign.CertificateProviderOptions{
			IDToken: idToken,
		}
	case cert != nil:
		bundleOpts.CertificateProvider = &localCertProvider{cert}
	default:
		publicKeyPem, err := keypair.GetPublicKeyPem()
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode([]byte(publicKeyPem))
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			log.Fatal(err)
		}
		verifier, err := signature.LoadDefaultVerifier(pubKey)
		if err != nil {
			log.Fatal(err)
		}
		key := root.NewExpiringKey(verifier, time.Time{}, time.Time{})
		keyTrustedMaterial := root.NewTrustedPublicKeyMaterial(func(_ string) (root.TimeConstrainedVerifier, error) {
			return key, nil
		})
		if bundleOpts.TrustedRoot != nil {
			trustedMaterial := &verifyTrustedMaterial{
				TrustedMaterial:    bundleOpts.TrustedRoot,
				keyTrustedMaterial: keyTrustedMaterial,
			}
			bundleOpts.TrustedRoot = trustedMaterial
		}
	}

	if len(signingConfig.TimestampAuthorityURLs()) != 0 {
		tsaSvcs, err := root.SelectServices(signingConfig.TimestampAuthorityURLs(),
			signingConfig.TimestampAuthorityURLsConfig(), sign.TimestampAuthorityAPIVersions, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		for _, tsaSvc := range tsaSvcs {
			tsaOpts := &sign.TimestampAuthorityOptions{
				URL:     tsaSvc.URL,
				Timeout: 30 * time.Second,
				Retries: 1,
			}
			if opts.TSAClientTransport != nil {
				tsaOpts.Transport = opts.TSAClientTransport
			}
			bundleOpts.TimestampAuthorities = append(bundleOpts.TimestampAuthorities, sign.NewTimestampAuthority(tsaOpts))
		}
	}

	var usingRekorV2 bool
	if len(signingConfig.RekorLogURLs()) != 0 {
		rekorSvcs, err := root.SelectServices(signingConfig.RekorLogURLs(),
			signingConfig.RekorLogURLsConfig(), sign.RekorAPIVersions, time.Now())
		if err != nil {
			return nil, err
		}
		for _, rekorSvc := range rekorSvcs {
			if rekorSvc.MajorAPIVersion == 2 {
				usingRekorV2 = true
			}
			rekorOpts := &sign.RekorOptions{
				BaseURL: rekorSvc.URL,
				Timeout: 90 * time.Second,
				Retries: 1,
				Version: rekorSvc.MajorAPIVersion,
			}
			bundleOpts.TransparencyLogs = append(bundleOpts.TransparencyLogs, sign.NewRekor(rekorOpts))
		}
	}
	// When requesting a short-lived Fulcio certificate, a timestamp must be provided during
	// verification. It can come from either Rekor v1 providing a signed entry timestamp, or
	// from a timestamp authority. Rekor v2 doesn't timestamp entries, so when a client
	// is configured to use Rekor v2 when retrieving a Fulcio certificate, we enforce
	// that a timestamp authority is provided as well.
	if usingRekorV2 && len(bundleOpts.TimestampAuthorities) == 0 && idToken != "" {
		return nil, fmt.Errorf("a timestamp authority must be provided to request a short-lived certificate that will be logged to Rekor")
	}

	spinner := ui.NewSpinner(ctx, "Signing artifact...")
	defer spinner.Stop()

	bundle, err := sign.Bundle(content, keypair, bundleOpts)

	if err != nil {
		return nil, fmt.Errorf("error signing bundle: %w", err)
	}
	return protojson.Marshal(bundle)
}

type verifyTrustedMaterial struct {
	root.TrustedMaterial
	keyTrustedMaterial root.TrustedMaterial
}

func (v *verifyTrustedMaterial) PublicKeyVerifier(hint string) (root.TimeConstrainedVerifier, error) {
	return v.keyTrustedMaterial.PublicKeyVerifier(hint)
}

type localCertProvider struct {
	cert []byte
}

func (c *localCertProvider) GetCertificate(_ context.Context, _ sign.Keypair, _ *sign.CertificateProviderOptions) ([]byte, error) {
	certBlock, _ := pem.Decode(c.cert)
	if certBlock == nil {
		return nil, fmt.Errorf("could not decode cert")
	}
	return certBlock.Bytes, nil
}

type cachingCertProvider struct {
	provider sign.CertificateProvider
	once     sync.Once
	fetch    func() ([]byte, error)
}

func (c *cachingCertProvider) GetCertificate(ctx context.Context, keypair sign.Keypair, opts *sign.CertificateProviderOptions) ([]byte, error) {
	c.once.Do(func() {
		c.fetch = sync.OnceValues(func() ([]byte, error) {
			return c.provider.GetCertificate(ctx, keypair, opts)
		})
	})
	return c.fetch()
}

func newFulcioProvider(signingConfig *root.SigningConfig) (sign.CertificateProvider, error) {
	if len(signingConfig.FulcioCertificateAuthorityURLs()) == 0 {
		return nil, fmt.Errorf("no fulcio URLs provided in signing config")
	}
	fulcioSvc, err := root.SelectService(signingConfig.FulcioCertificateAuthorityURLs(), sign.FulcioAPIVersions, time.Now())
	if err != nil {
		return nil, err
	}
	fulcioOpts := &sign.FulcioOptions{
		BaseURL: fulcioSvc.URL,
		Timeout: 30 * time.Second,
		Retries: 1,
	}
	return sign.NewFulcio(fulcioOpts), nil
}

// NewCachingFulcioProvider creates a caching Fulcio provider from the given signing config.
// This function should not be used in long-running processes, as the certificate will
// expire.
func NewCachingFulcioProvider(signingConfig *root.SigningConfig) (sign.CertificateProvider, error) {
	provider, err := newFulcioProvider(signingConfig)
	if err != nil {
		return nil, err
	}
	return &cachingCertProvider{provider: provider}, nil
}
