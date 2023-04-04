// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crl_x509

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/test"
)

const (
	derCRLBase64 = "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"
	// NOTE: This constant does not exist in upstream.
	derGenTimeZ0000Base64 = "MIIBSzCB0wIBATAKBggqhkjOPQQDAzBJMQswCQYDVQQGEwJYWDEVMBMGA1UEChMMQm91bGRlciBUZXN0MSMwIQYDVQQDExooVEVTVCkgRWxlZ2FudCBFbGVwaGFudCBFMRgTMjA1MDA3MDYxNjQzMzhaMDAwMBcNMjIwNzE1MTY0MzM4WjAbMBkCCAOuUdtRFVo8Fw0yMjA3MDYxNTQzMzhaoDYwNDAfBgNVHSMEGDAWgBQB2rt6yyUgjl551vmWQi8CQSkHvjARBgNVHRQECgIIFv9LJt+yGA8wCgYIKoZIzj0EAwMDZwAwZAIwVrITRYutGjFpfNht08CLsAQSvnc4i6UM0Pi8+U3T8DRHImIiuB9cQ+qxULB6pKhBAjBbuGCwTop7vCfGO7Fz6N0ruITInFtt6BDR5izWUMfXXa7mXhSQ6ig9hOHOWRxR00I="
	// NOTE: This constant does not exist in upstream.
	derUTCTimeYYMMDDHHMMZTTBase64 = "MIIBRTCBzQIBATAKBggqhkjOPQQDAzBJMQswCQYDVQQGEwJYWDEVMBMGA1UEChMMQm91bGRlciBUZXN0MSMwIQYDVQQDExooVEVTVCkgRWxlZ2FudCBFbGVwaGFudCBFMRcNMjIwNzA2MTY0M1owOBcNMjIwNzE1MTY0MzM4WjAbMBkCCAOuUdtRFVo8Fw0yMjA3MDYxNTQzMzhaoDYwNDAfBgNVHSMEGDAWgBQB2rt6yyUgjl551vmWQi8CQSkHvjARBgNVHRQECgIIFv9LJt+yGA8wCgYIKoZIzj0EAwMDZwAwZAIwVrITRYutGjFpfNht08CLsAQSvnc4i6UM0Pi8+U3T8DRHImIiuB9cQ+qxULB6pKhBAjBbuGCwTop7vCfGO7Fz6N0ruITInFtt6BDR5izWUMfXXa7mXhSQ6ig9hOHOWRxR00I="
)

func TestCreateRevocationList(t *testing.T) {
	ec256Priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA P256 key: %s", err)
	}
	_, ed25519Priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %s", err)
	}
	reasonKeyCompromise := 1
	tests := []struct {
		name          string
		key           crypto.Signer
		issuer        *x509.Certificate
		template      *RevocationList
		expectedError string
	}{
		{
			name:          "nil template",
			key:           ec256Priv,
			issuer:        nil,
			template:      nil,
			expectedError: "x509: template can not be nil",
		},
		{
			name:          "nil issuer",
			key:           ec256Priv,
			issuer:        nil,
			template:      &RevocationList{},
			expectedError: "x509: issuer can not be nil",
		},
		{
			name: "issuer doesn't have crlSign key usage bit set",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCertSign,
			},
			template:      &RevocationList{},
			expectedError: "x509: issuer must have the crlSign key usage bit set",
		},
		{
			name: "issuer missing SubjectKeyId",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
			},
			template:      &RevocationList{},
			expectedError: "x509: issuer certificate doesn't contain a subject key identifier",
		},
		{
			name: "nextUpdate before thisUpdate",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				ThisUpdate: time.Time{}.Add(time.Hour),
				NextUpdate: time.Time{},
			},
			expectedError: "x509: template.ThisUpdate is after template.NextUpdate",
		},
		{
			name: "nil Number",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
			expectedError: "x509: template contains nil Number field",
		},
		{
			name: "invalid signature algorithm",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				SignatureAlgorithm: SHA256WithRSA,
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
			expectedError: "x509: requested SignatureAlgorithm does not match private key type",
		},
		{
			name: "long Number",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
				Number:     big.NewInt(0).SetBytes(append([]byte{1}, make([]byte, 20)...)),
			},
			expectedError: "x509: CRL number exceeds 20 octets",
		},
		{
			name: "long Number (20 bytes, MSB set)",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
				Number:     big.NewInt(0).SetBytes(append([]byte{255}, make([]byte, 19)...)),
			},
			expectedError: "x509: CRL number exceeds 20 octets",
		},
		{
			name: "valid",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
		{
			name: "valid, reason code",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
						ReasonCode:     &reasonKeyCompromise,
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
		{
			name: "valid, extra entry extension",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
						ReasonCode:     &reasonKeyCompromise,
						ExtraExtensions: []pkix.Extension{
							{
								Id:    []int{1, 1},
								Value: []byte{5, 0},
							},
						},
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
		{
			name: "valid, Ed25519 key",
			key:  ed25519Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
		{
			name: "valid, non-default signature algorithm",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				SignatureAlgorithm: ECDSAWithSHA512,
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
		{
			name: "valid, extra extension",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				RevokedCertificates: []RevokedCertificate{
					{
						SerialNumber:   big.NewInt(2),
						RevocationTime: time.Time{}.Add(time.Hour),
					},
				},
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
				ExtraExtensions: []pkix.Extension{
					{
						Id:    []int{2, 5, 29, 99},
						Value: []byte{5, 0},
					},
				},
			},
		},
		{
			name: "valid, empty list",
			key:  ec256Priv,
			issuer: &x509.Certificate{
				KeyUsage: x509.KeyUsageCRLSign,
				Subject: pkix.Name{
					CommonName: "testing",
				},
				SubjectKeyId: []byte{1, 2, 3},
			},
			template: &RevocationList{
				Number:     big.NewInt(5),
				ThisUpdate: time.Time{}.Add(time.Hour * 24),
				NextUpdate: time.Time{}.Add(time.Hour * 48),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			crl, err := CreateRevocationList(rand.Reader, tc.template, tc.issuer, tc.key)
			if err != nil && tc.expectedError == "" {
				t.Fatalf("CreateRevocationList failed unexpectedly: %s", err)
			} else if err != nil && tc.expectedError != err.Error() {
				t.Fatalf("CreateRevocationList failed unexpectedly, wanted: %s, got: %s", tc.expectedError, err)
			} else if err == nil && tc.expectedError != "" {
				t.Fatalf("CreateRevocationList didn't fail, expected: %s", tc.expectedError)
			}
			if tc.expectedError != "" {
				return
			}

			parsedCRL, err := ParseRevocationList(crl)
			if err != nil {
				t.Fatalf("Failed to parse generated CRL: %s", err)
			}

			if tc.template.SignatureAlgorithm != UnknownSignatureAlgorithm &&
				parsedCRL.SignatureAlgorithm != tc.template.SignatureAlgorithm {
				t.Fatalf("SignatureAlgorithm mismatch: got %v; want %v.", parsedCRL.SignatureAlgorithm,
					tc.template.SignatureAlgorithm)
			}

			if len(parsedCRL.RevokedCertificates) != len(tc.template.RevokedCertificates) {
				t.Fatalf("RevokedCertificates length mismatch: got %d; want %d.",
					len(parsedCRL.RevokedCertificates), len(tc.template.RevokedCertificates))
			}
			for i, rc := range parsedCRL.RevokedCertificates {
				erc := tc.template.RevokedCertificates[i]
				if rc.SerialNumber.Cmp(erc.SerialNumber) != 0 {
					t.Errorf("RevokedCertificates entry %d serial mismatch: got %s; want %s.",
						i, rc.SerialNumber.String(), erc.SerialNumber.String())
				}
				if rc.RevocationTime != erc.RevocationTime {
					t.Errorf("RevokedCertificates entry %d date mismatch: got %v; want %v.",
						i, rc.RevocationTime, erc.RevocationTime)
				}
				numExtra := 0
				if erc.ReasonCode != nil {
					if rc.ReasonCode == nil {
						t.Errorf("RevokedCertificates entry %d reason mismatch: got nil; want %v.",
							i, *erc.ReasonCode)
					}
					if *rc.ReasonCode != *erc.ReasonCode {
						t.Errorf("RevokedCertificates entry %d reason mismatch: got %v; want %v.",
							i, *rc.ReasonCode, *erc.ReasonCode)
					}
					numExtra = 1
				} else {
					if rc.ReasonCode != nil {
						t.Errorf("RevokedCertificates entry %d reason mismatch: got %v; want nil.",
							i, *rc.ReasonCode)
					}
				}
				if len(rc.Extensions) != numExtra+len(erc.ExtraExtensions) {
					t.Errorf("RevokedCertificates entry %d has wrong number of extensions: got %d; want %d",
						i, len(rc.Extensions), numExtra+len(erc.ExtraExtensions))
				}
			}

			if len(parsedCRL.Extensions) != 2+len(tc.template.ExtraExtensions) {
				t.Fatalf("Generated CRL has wrong number of extensions, wanted: %d, got: %d", 2+len(tc.template.ExtraExtensions), len(parsedCRL.Extensions))
			}
			expectedAKI, err := asn1.Marshal(authKeyId{Id: tc.issuer.SubjectKeyId})
			if err != nil {
				t.Fatalf("asn1.Marshal failed: %s", err)
			}
			akiExt := pkix.Extension{
				Id:    oidExtensionAuthorityKeyId,
				Value: expectedAKI,
			}
			if !reflect.DeepEqual(parsedCRL.Extensions[0], akiExt) {
				t.Fatalf("Unexpected first extension: got %v, want %v",
					parsedCRL.Extensions[0], akiExt)
			}
			expectedNum, err := asn1.Marshal(tc.template.Number)
			if err != nil {
				t.Fatalf("asn1.Marshal failed: %s", err)
			}
			crlExt := pkix.Extension{
				Id:    oidExtensionCRLNumber,
				Value: expectedNum,
			}
			if !reflect.DeepEqual(parsedCRL.Extensions[1], crlExt) {
				t.Fatalf("Unexpected second extension: got %v, want %v",
					parsedCRL.Extensions[1], crlExt)
			}
			if len(parsedCRL.Extensions[2:]) == 0 && len(tc.template.ExtraExtensions) == 0 {
				// If we don't have anything to check return early so we don't
				// hit a [] != nil false positive below.
				return
			}
			if !reflect.DeepEqual(parsedCRL.Extensions[2:], tc.template.ExtraExtensions) {
				t.Fatalf("Extensions mismatch: got %v; want %v.",
					parsedCRL.Extensions[2:], tc.template.ExtraExtensions)
			}
		})
	}
}

func TestParseRevocationList(t *testing.T) {
	derBytes := fromBase64(derCRLBase64)
	certList, err := ParseRevocationList(derBytes)
	if err != nil {
		t.Errorf("error parsing: %s", err)
		return
	}
	numCerts := len(certList.RevokedCertificates)
	expected := 88
	if numCerts != expected {
		t.Errorf("bad number of revoked certificates. got: %d want: %d", numCerts, expected)
	}

	// NOTE: This block does not exist in upstream.
	// Check that 'thisUpdate' of 'GENERALIZEDTIME 20500706164338Z0000' (time
	// zone is UTC but also explicitly specified) is considered invalid.
	derBytes = fromBase64(derGenTimeZ0000Base64)
	_, err = ParseRevocationList(derBytes)
	test.AssertError(t, err, "expected error parsing CRL")
	test.AssertContains(t, err.Error(), "x509: malformed GeneralizedTime")

	// NOTE: This block does not exist in upstream.
	// Check that 'thisUpdate' of 'UTCTIME 2207061643Z08' (YYMMDDHHMMZTT) is
	// considered invalid.
	derBytes = fromBase64(derUTCTimeYYMMDDHHMMZTTBase64)
	_, err = ParseRevocationList(derBytes)
	test.AssertContains(t, err.Error(), "x509: malformed UTCTime")
}

func TestRevocationListCheckSignatureFrom(t *testing.T) {
	goodKey, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %s", err)
	}
	badKey, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %s", err)
	}
	tests := []struct {
		name   string
		issuer *x509.Certificate
		err    string
	}{
		{
			name: "valid",
			issuer: &x509.Certificate{
				Version:               3,
				BasicConstraintsValid: true,
				IsCA:                  true,
				PublicKeyAlgorithm:    x509.ECDSA,
				PublicKey:             goodKey.Public(),
			},
		},
		{
			name: "valid, key usage set",
			issuer: &x509.Certificate{
				Version:               3,
				BasicConstraintsValid: true,
				IsCA:                  true,
				PublicKeyAlgorithm:    x509.ECDSA,
				PublicKey:             goodKey.Public(),
				KeyUsage:              x509.KeyUsageCRLSign,
			},
		},
		{
			name: "invalid issuer, wrong key usage",
			issuer: &x509.Certificate{
				Version:               3,
				BasicConstraintsValid: true,
				IsCA:                  true,
				PublicKeyAlgorithm:    x509.ECDSA,
				PublicKey:             goodKey.Public(),
				KeyUsage:              x509.KeyUsageCertSign,
			},
			err: "x509: invalid signature: parent certificate cannot sign this kind of certificate",
		},
		{
			name: "invalid issuer, no basic constraints/ca",
			issuer: &x509.Certificate{
				Version:            3,
				PublicKeyAlgorithm: x509.ECDSA,
				PublicKey:          goodKey.Public(),
			},
			err: "x509: invalid signature: parent certificate cannot sign this kind of certificate",
		},
		{
			name: "invalid issuer, unsupported public key type",
			issuer: &x509.Certificate{
				Version:               3,
				BasicConstraintsValid: true,
				IsCA:                  true,
				PublicKeyAlgorithm:    x509.UnknownPublicKeyAlgorithm,
				PublicKey:             goodKey.Public(),
			},
			err: "x509: cannot verify signature: algorithm unimplemented",
		},
		{
			name: "wrong key",
			issuer: &x509.Certificate{
				Version:               3,
				BasicConstraintsValid: true,
				IsCA:                  true,
				PublicKeyAlgorithm:    x509.ECDSA,
				PublicKey:             badKey.Public(),
			},
			err: "x509: ECDSA verification failure",
		},
	}

	crlIssuer := &x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		PublicKeyAlgorithm:    x509.ECDSA,
		PublicKey:             goodKey.Public(),
		KeyUsage:              x509.KeyUsageCRLSign,
		SubjectKeyId:          []byte{1, 2, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			crlDER, err := CreateRevocationList(rand.Reader, &RevocationList{Number: big.NewInt(1)}, crlIssuer, goodKey)
			if err != nil {
				t.Fatalf("failed to generate CRL: %s", err)
			}
			crl, err := ParseRevocationList(crlDER)
			if err != nil {
				t.Fatalf("failed to parse test CRL: %s", err)
			}
			err = crl.CheckSignatureFrom(tc.issuer)
			if err != nil && err.Error() != tc.err {
				t.Errorf("unexpected error: got %s, want %s", err, tc.err)
			} else if err == nil && tc.err != "" {
				t.Errorf("CheckSignatureFrom did not fail: want %s", tc.err)
			}
		})
	}
}
