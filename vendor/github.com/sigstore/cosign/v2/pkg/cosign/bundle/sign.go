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
	"time"

	"github.com/sigstore/cosign/v2/internal/ui"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/sign"
	"github.com/sigstore/sigstore/pkg/signature"
	"google.golang.org/protobuf/encoding/protojson"
)

func SignData(ctx context.Context, content sign.Content, keypair sign.Keypair, idToken string, signingConfig *root.SigningConfig, trustedMaterial root.TrustedMaterial) ([]byte, error) {
	var opts sign.BundleOptions

	if trustedMaterial != nil {
		opts.TrustedRoot = trustedMaterial
	}

	if idToken != "" {
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
		opts.CertificateProvider = sign.NewFulcio(fulcioOpts)
		opts.CertificateProviderOptions = &sign.CertificateProviderOptions{
			IDToken: idToken,
		}
	} else {
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
		trustedMaterial := &verifyTrustedMaterial{
			TrustedMaterial:    opts.TrustedRoot,
			keyTrustedMaterial: keyTrustedMaterial,
		}
		opts.TrustedRoot = trustedMaterial
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
			opts.TimestampAuthorities = append(opts.TimestampAuthorities, sign.NewTimestampAuthority(tsaOpts))
		}
	}

	if len(signingConfig.RekorLogURLs()) != 0 {
		rekorSvcs, err := root.SelectServices(signingConfig.RekorLogURLs(),
			signingConfig.RekorLogURLsConfig(), sign.RekorAPIVersions, time.Now())
		if err != nil {
			return nil, err
		}
		for _, rekorSvc := range rekorSvcs {
			rekorOpts := &sign.RekorOptions{
				BaseURL: rekorSvc.URL,
				Timeout: 90 * time.Second,
				Retries: 1,
				Version: rekorSvc.MajorAPIVersion,
			}
			opts.TransparencyLogs = append(opts.TransparencyLogs, sign.NewRekor(rekorOpts))
		}
	}

	spinner := ui.NewSpinner(ctx, "Signing artifact...")
	defer spinner.Stop()

	bundle, err := sign.Bundle(content, keypair, opts)

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
