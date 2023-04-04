// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package crl_x509 parses X.509-encoded certificate revocation lists.
package crl_x509

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
)

var (
	oidExtensionReasonCode = []int{2, 5, 29, 21}
)

// RevokedCertificate represents an entry in the revokedCertificates sequence of
// a CRL.
// NOTE: This type does not exist in upstream.
type RevokedCertificate struct {
	// Raw contains the raw bytes of the revokedCertificates entry. It is set when
	// parsing a CRL; it is ignored when generating a CRL.
	Raw []byte

	// SerialNumber represents the serial number of a revoked certificate. It is
	// both used when creating a CRL and populated when parsing a CRL. It MUST NOT
	// be nil.
	SerialNumber *big.Int
	// RevocationTime represents the time at which the certificate was revoked. It
	// is both used when creating a CRL and populated when parsing a CRL. It MUST
	// NOT be nil.
	RevocationTime time.Time
	// ReasonCode represents the reason for revocation, using the integer enum
	// values specified in RFC 5280 Section 5.3.1. When creating a CRL, a value of
	// nil or zero will result in the reasonCode extension being omitted. When
	// parsing a CRL, a value of nil represents a no reasonCode extension, while a
	// value of 0 represents a reasonCode extension containing enum value 0 (this
	// SHOULD NOT happen, but can and does).
	ReasonCode *int

	// Extensions contains raw X.509 extensions. When creating a CRL, the
	// Extensions field is ignored, see ExtraExtensions.
	Extensions []pkix.Extension
	// ExtraExtensions contains any additional extensions to add directly to the
	// revokedCertificate entry. It is up to the caller to ensure that this field
	// does not contain any extensions which duplicate extensions created by this
	// package (currently, the reasonCode extension). The ExtraExtensions field is
	// not populated when parsing a CRL, see Extensions.
	ExtraExtensions []pkix.Extension
}

// RevocationList represents a Certificate Revocation List (CRL) as specified
// by RFC 5280.
type RevocationList struct {
	// Raw, RawTBSRevocationList, and RawIssuer contain the raw bytes of the whole
	// CRL, the tbsCertList field, and the issuer field, respectively. They are
	// populated when parsing a CRL; they are ignored when creating a CRL.
	Raw                  []byte
	RawTBSRevocationList []byte
	RawIssuer            []byte

	// Issuer is the name of the issuer which issued the CRL. It is ignored when
	// creating a CRL.
	Issuer pkix.Name

	// Signature is the signature contained within the CRL. It is ignored when
	// creating a CRL.
	Signature []byte
	// SignatureAlgorithm is the signature algorithm used when signing the CRL.
	// When creating a CRL, a value of 0 means that the default algorithm for the
	// signing key will be used.
	SignatureAlgorithm SignatureAlgorithm

	// ThisUpdate represents the thisUpdate field in the CRL, which indicates the
	// issuance date of the CRL. It is both used when creating a CRL and populated
	// when parsing a CRL.
	ThisUpdate time.Time
	// NextUpdate represents the nextUpdate field in the CRL, which indicates the
	// date by which the next CRL will be issued. NextUpdate must be greater than
	// ThisUpdate. It is both used when creating a CRL and populated when parsing
	// a CRL.
	NextUpdate time.Time

	// RevokedCertificates represents the revokedCertificates sequence in the CRL.
	// It is both used when creating a CRL and populated when parsing a CRL. When
	// creating a CRL, it may be empty or nil, in which case the
	// revokedCertificates sequence will be omitted from the CRL entirely.
	RevokedCertificates []RevokedCertificate

	// Number represents the CRLNumber extension, which should be a monotonically
	// increasing sequence number for a given CRL scope and CRL issuer. It is both
	// used when creating a CRL and populated when parsing a CRL. When creating a
	// CRL, it MUST NOT be nil, and MUST NOT be longer than 20 bytes.
	Number *big.Int

	// AuthorityKeyId is populated from the authorityKeyIdentifier extension in
	// the CRL. It is ignored when creating a CRL: the extension is populated from
	// the issuer information instead.
	AuthorityKeyId []byte

	// Extensions contains raw X.509 extensions. When creating a CRL, the
	// Extensions field is ignored, see ExtraExtensions.
	Extensions []pkix.Extension
	// ExtraExtensions contains any additional extensions to add directly to the
	// CRL. It is up to the caller to ensure that this field does not contain any
	// extensions which duplicate extensions created by this package (currently,
	// the number and authorityKeyIdentifier extensions). The ExtraExtensions
	// field is not populated when parsing a CRL, see Extensions.
	ExtraExtensions []pkix.Extension
}

// ParseRevocationList parses a X509 v2 Certificate Revocation List from the given
// ASN.1 DER data.
func ParseRevocationList(der []byte) (*RevocationList, error) {
	rl := &RevocationList{}

	input := cryptobyte.String(der)
	// we read the SEQUENCE including length and tag bytes so that
	// we can populate RevocationList.Raw, before unwrapping the
	// SEQUENCE so it can be operated on
	if !input.ReadASN1Element(&input, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed crl")
	}
	rl.Raw = input
	if !input.ReadASN1(&input, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed crl")
	}

	var tbs cryptobyte.String
	// do the same trick again as above to extract the raw
	// bytes for Certificate.RawTBSCertificate
	if !input.ReadASN1Element(&tbs, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed tbs crl")
	}
	rl.RawTBSRevocationList = tbs
	if !tbs.ReadASN1(&tbs, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed tbs crl")
	}

	var version int
	if !tbs.PeekASN1Tag(cryptobyte_asn1.INTEGER) {
		return nil, errors.New("x509: unsupported crl version")
	}
	if !tbs.ReadASN1Integer(&version) {
		return nil, errors.New("x509: malformed crl")
	}
	if version != x509v2Version {
		return nil, fmt.Errorf("x509: unsupported crl version: %d", version)
	}

	var sigAISeq cryptobyte.String
	if !tbs.ReadASN1(&sigAISeq, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed signature algorithm identifier")
	}
	// Before parsing the inner algorithm identifier, extract
	// the outer algorithm identifier and make sure that they
	// match.
	var outerSigAISeq cryptobyte.String
	if !input.ReadASN1(&outerSigAISeq, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed algorithm identifier")
	}
	if !bytes.Equal(outerSigAISeq, sigAISeq) {
		return nil, errors.New("x509: inner and outer signature algorithm identifiers don't match")
	}
	sigAI, err := parseAI(sigAISeq)
	if err != nil {
		return nil, err
	}
	rl.SignatureAlgorithm = getSignatureAlgorithmFromAI(sigAI)

	var signature asn1.BitString
	if !input.ReadASN1BitString(&signature) {
		return nil, errors.New("x509: malformed signature")
	}
	rl.Signature = signature.RightAlign()

	var issuerSeq cryptobyte.String
	if !tbs.ReadASN1Element(&issuerSeq, cryptobyte_asn1.SEQUENCE) {
		return nil, errors.New("x509: malformed issuer")
	}
	rl.RawIssuer = issuerSeq
	issuerRDNs, err := parseName(issuerSeq)
	if err != nil {
		return nil, err
	}
	rl.Issuer.FillFromRDNSequence(issuerRDNs)

	rl.ThisUpdate, err = parseTime(&tbs)
	if err != nil {
		return nil, err
	}
	if tbs.PeekASN1Tag(cryptobyte_asn1.GeneralizedTime) || tbs.PeekASN1Tag(cryptobyte_asn1.UTCTime) {
		rl.NextUpdate, err = parseTime(&tbs)
		if err != nil {
			return nil, err
		}
	}

	if tbs.PeekASN1Tag(cryptobyte_asn1.SEQUENCE) {
		rcs := make([]RevokedCertificate, 0)
		var revokedSeq cryptobyte.String
		if !tbs.ReadASN1(&revokedSeq, cryptobyte_asn1.SEQUENCE) {
			return nil, errors.New("x509: malformed crl")
		}
		for !revokedSeq.Empty() {
			var certSeq cryptobyte.String
			if !revokedSeq.ReadASN1Element(&certSeq, cryptobyte_asn1.SEQUENCE) {
				return nil, errors.New("x509: malformed crl")
			}
			rc := RevokedCertificate{Raw: certSeq}
			if !certSeq.ReadASN1(&certSeq, cryptobyte_asn1.SEQUENCE) {
				return nil, errors.New("x509: malformed crl")
			}
			rc.SerialNumber = new(big.Int)
			if !certSeq.ReadASN1Integer(rc.SerialNumber) {
				return nil, errors.New("x509: malformed serial number")
			}
			rc.RevocationTime, err = parseTime(&certSeq)
			if err != nil {
				return nil, err
			}
			var extensions cryptobyte.String
			var present bool
			if !certSeq.ReadOptionalASN1(&extensions, &present, cryptobyte_asn1.SEQUENCE) {
				return nil, errors.New("x509: malformed extensions")
			}
			if present {
				for !extensions.Empty() {
					var extension cryptobyte.String
					if !extensions.ReadASN1(&extension, cryptobyte_asn1.SEQUENCE) {
						return nil, errors.New("x509: malformed extension")
					}
					ext, err := parseExtension(extension)
					if err != nil {
						return nil, err
					}
					// NOTE: This block does not exist in upstream.
					if ext.Id.Equal(oidExtensionReasonCode) {
						val := cryptobyte.String(ext.Value)
						rc.ReasonCode = new(int)
						if !val.ReadASN1Enum(rc.ReasonCode) {
							return nil, fmt.Errorf("x509: malformed reasonCode extension")
						}
					}
					rc.Extensions = append(rc.Extensions, ext)
				}
			}

			rcs = append(rcs, rc)
		}
		rl.RevokedCertificates = rcs
	}

	var extensions cryptobyte.String
	var present bool
	if !tbs.ReadOptionalASN1(&extensions, &present, cryptobyte_asn1.Tag(0).Constructed().ContextSpecific()) {
		return nil, errors.New("x509: malformed extensions")
	}
	if present {
		if !extensions.ReadASN1(&extensions, cryptobyte_asn1.SEQUENCE) {
			return nil, errors.New("x509: malformed extensions")
		}
		for !extensions.Empty() {
			var extension cryptobyte.String
			if !extensions.ReadASN1(&extension, cryptobyte_asn1.SEQUENCE) {
				return nil, errors.New("x509: malformed extension")
			}
			ext, err := parseExtension(extension)
			if err != nil {
				return nil, err
			}
			// NOTE: The block does not exist in upstream.
			if ext.Id.Equal(oidExtensionAuthorityKeyId) {
				rl.AuthorityKeyId = ext.Value
			} else if ext.Id.Equal(oidExtensionCRLNumber) {
				value := cryptobyte.String(ext.Value)
				rl.Number = new(big.Int)
				if !value.ReadASN1Integer(rl.Number) {
					return nil, errors.New("x509: malformed crl number")
				}
			}
			rl.Extensions = append(rl.Extensions, ext)
		}
	}

	return rl, nil
}

// CreateRevocationList creates a new X.509 v2 Certificate Revocation List,
// according to RFC 5280, based on template.
//
// The CRL is signed by priv which should be the private key associated with
// the public key in the issuer certificate.
//
// The issuer may not be nil, and the crlSign bit must be set in KeyUsage in
// order to use it as a CRL issuer.
//
// The issuer distinguished name CRL field and authority key identifier
// extension are populated using the issuer certificate. issuer must have
// SubjectKeyId set.
func CreateRevocationList(rand io.Reader, template *RevocationList, issuer *x509.Certificate, priv crypto.Signer) ([]byte, error) {
	if template == nil {
		return nil, errors.New("x509: template can not be nil")
	}
	if issuer == nil {
		return nil, errors.New("x509: issuer can not be nil")
	}
	if (issuer.KeyUsage & x509.KeyUsageCRLSign) == 0 {
		return nil, errors.New("x509: issuer must have the crlSign key usage bit set")
	}
	if len(issuer.SubjectKeyId) == 0 {
		return nil, errors.New("x509: issuer certificate doesn't contain a subject key identifier")
	}
	if template.NextUpdate.Before(template.ThisUpdate) {
		return nil, errors.New("x509: template.ThisUpdate is after template.NextUpdate")
	}
	if template.Number == nil {
		return nil, errors.New("x509: template contains nil Number field")
	}

	hashFunc, signatureAlgorithm, err := signingParamsForPublicKey(priv.Public(), template.SignatureAlgorithm)
	if err != nil {
		return nil, err
	}

	// Convert the ReasonCode field to a proper extension, and force revocation
	// times to UTC per RFC 5280.
	// NOTE: This block differs from upstream.
	revokedCerts := make([]pkix.RevokedCertificate, len(template.RevokedCertificates))
	for i, rc := range template.RevokedCertificates {
		prc := pkix.RevokedCertificate{
			SerialNumber:   rc.SerialNumber,
			RevocationTime: rc.RevocationTime.UTC(),
		}

		// Copy over any extra extensions, except for a Reason Code extension,
		// because we'll synthesize that ourselves to ensure it is correct.
		exts := make([]pkix.Extension, 0)
		for _, ext := range rc.ExtraExtensions {
			if ext.Id.Equal(oidExtensionReasonCode) {
				continue
			}
			exts = append(exts, ext)
		}

		// Only add a reasonCode extension if the reason is non-zero, as per
		// RFC 5280 Section 5.3.1.
		if rc.ReasonCode != nil && *rc.ReasonCode != 0 {
			reasonBytes, err := asn1.Marshal(asn1.Enumerated(*rc.ReasonCode))
			if err != nil {
				return nil, err
			}

			exts = append(exts, pkix.Extension{
				Id:    oidExtensionReasonCode,
				Value: reasonBytes,
			})
		}

		if len(exts) > 0 {
			prc.Extensions = exts
		}
		revokedCerts[i] = prc
	}

	aki, err := asn1.Marshal(authKeyId{Id: issuer.SubjectKeyId})
	if err != nil {
		return nil, err
	}

	if numBytes := template.Number.Bytes(); len(numBytes) > 20 || (len(numBytes) == 20 && numBytes[0]&0x80 != 0) {
		return nil, errors.New("x509: CRL number exceeds 20 octets")
	}
	crlNum, err := asn1.Marshal(template.Number)
	if err != nil {
		return nil, err
	}

	tbsCertList := pkix.TBSCertificateList{
		Version:    1, // v2
		Signature:  signatureAlgorithm,
		Issuer:     issuer.Subject.ToRDNSequence(),
		ThisUpdate: template.ThisUpdate.UTC(),
		NextUpdate: template.NextUpdate.UTC(),
		Extensions: []pkix.Extension{
			{
				Id:    oidExtensionAuthorityKeyId,
				Value: aki,
			},
			{
				Id:    oidExtensionCRLNumber,
				Value: crlNum,
			},
		},
	}
	if len(revokedCerts) > 0 {
		tbsCertList.RevokedCertificates = revokedCerts
	}

	if len(template.ExtraExtensions) > 0 {
		tbsCertList.Extensions = append(tbsCertList.Extensions, template.ExtraExtensions...)
	}

	tbsCertListContents, err := asn1.Marshal(tbsCertList)
	if err != nil {
		return nil, err
	}

	input := tbsCertListContents
	if hashFunc != 0 {
		h := hashFunc.New()
		h.Write(tbsCertListContents)
		input = h.Sum(nil)
	}
	var signerOpts crypto.SignerOpts = hashFunc
	if template.SignatureAlgorithm.isRSAPSS() {
		signerOpts = &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       hashFunc,
		}
	}

	signature, err := priv.Sign(rand, input, signerOpts)
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(pkix.CertificateList{
		TBSCertList:        tbsCertList,
		SignatureAlgorithm: signatureAlgorithm,
		SignatureValue:     asn1.BitString{Bytes: signature, BitLength: len(signature) * 8},
	})
}

// CheckSignatureFrom verifies that the signature on rl is a valid signature
// from issuer.
func (rl *RevocationList) CheckSignatureFrom(parent *x509.Certificate) error {
	if parent.Version == 3 && !parent.BasicConstraintsValid ||
		parent.BasicConstraintsValid && !parent.IsCA {
		return x509.ConstraintViolationError{}
	}

	if parent.KeyUsage != 0 && parent.KeyUsage&x509.KeyUsageCRLSign == 0 {
		return x509.ConstraintViolationError{}
	}

	if parent.PublicKeyAlgorithm == x509.UnknownPublicKeyAlgorithm {
		return x509.ErrUnsupportedAlgorithm
	}

	return parent.CheckSignature(x509.SignatureAlgorithm(rl.SignatureAlgorithm), rl.RawTBSRevocationList, rl.Signature)
}
