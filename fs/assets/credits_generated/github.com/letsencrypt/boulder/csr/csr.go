package csr

import (
	"context"
	"crypto"
	"crypto/x509"
	"errors"
	"strings"

	"github.com/letsencrypt/boulder/core"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/goodkey"
	"github.com/letsencrypt/boulder/identifier"
)

// maxCNLength is the maximum length allowed for the common name as specified in RFC 5280
const maxCNLength = 64

// This map is used to decide which CSR signing algorithms we consider
// strong enough to use. Significantly the missing algorithms are:
// * No algorithms using MD2, MD5, or SHA-1
// * No DSA algorithms
var goodSignatureAlgorithms = map[x509.SignatureAlgorithm]bool{
	x509.SHA256WithRSA:   true,
	x509.SHA384WithRSA:   true,
	x509.SHA512WithRSA:   true,
	x509.ECDSAWithSHA256: true,
	x509.ECDSAWithSHA384: true,
	x509.ECDSAWithSHA512: true,
}

var (
	invalidPubKey        = berrors.BadCSRError("invalid public key in CSR")
	unsupportedSigAlg    = berrors.BadCSRError("signature algorithm not supported")
	invalidSig           = berrors.BadCSRError("invalid signature on CSR")
	invalidEmailPresent  = berrors.BadCSRError("CSR contains one or more email address fields")
	invalidIPPresent     = berrors.BadCSRError("CSR contains one or more IP address fields")
	invalidNoDNS         = berrors.BadCSRError("at least one DNS name is required")
	invalidAllSANTooLong = berrors.BadCSRError("CSR doesn't contain a SAN short enough to fit in CN")
)

// VerifyCSR checks the validity of a x509.CertificateRequest. Before doing checks it normalizes
// the CSR which lowers the case of DNS names and subject CN, and hoist a DNS name into the CN
// if it is empty.
func VerifyCSR(ctx context.Context, csr *x509.CertificateRequest, maxNames int, keyPolicy *goodkey.KeyPolicy, pa core.PolicyAuthority) error {
	normalizeCSR(csr)
	key, ok := csr.PublicKey.(crypto.PublicKey)
	if !ok {
		return invalidPubKey
	}
	err := keyPolicy.GoodKey(ctx, key)
	if err != nil {
		if errors.Is(err, goodkey.ErrBadKey) {
			return berrors.BadCSRError("invalid public key in CSR: %s", err)
		}
		return berrors.InternalServerError("error checking key validity: %s", err)
	}
	if !goodSignatureAlgorithms[csr.SignatureAlgorithm] {
		return unsupportedSigAlg
	}

	err = csr.CheckSignature()
	if err != nil {
		return invalidSig
	}
	if len(csr.EmailAddresses) > 0 {
		return invalidEmailPresent
	}
	if len(csr.IPAddresses) > 0 {
		return invalidIPPresent
	}
	if len(csr.DNSNames) == 0 && csr.Subject.CommonName == "" {
		return invalidNoDNS
	}
	if csr.Subject.CommonName == "" {
		return invalidAllSANTooLong
	}
	if len(csr.Subject.CommonName) > maxCNLength {
		return berrors.BadCSRError("CN was longer than %d bytes", maxCNLength)
	}
	if len(csr.DNSNames) > maxNames {
		return berrors.BadCSRError("CSR contains more than %d DNS names", maxNames)
	}
	idents := make([]identifier.ACMEIdentifier, len(csr.DNSNames))
	for i, dnsName := range csr.DNSNames {
		idents[i] = identifier.DNSIdentifier(dnsName)
	}
	err = pa.WillingToIssueWildcards(idents)
	if err != nil {
		return err
	}
	return nil
}

// normalizeCSR deduplicates and lowers the case of dNSNames and the subject CN.
// It will also hoist a dNSName into the CN if it is empty.
func normalizeCSR(csr *x509.CertificateRequest) {
	if csr.Subject.CommonName == "" {
		var forcedCN string
		// Promote the first SAN that is less than maxCNLength (if any)
		for _, name := range csr.DNSNames {
			if len(name) <= maxCNLength {
				forcedCN = name
				break
			}
		}
		csr.Subject.CommonName = forcedCN
	} else if csr.Subject.CommonName != "" {
		csr.DNSNames = append(csr.DNSNames, csr.Subject.CommonName)
	}
	csr.Subject.CommonName = strings.ToLower(csr.Subject.CommonName)
	csr.DNSNames = core.UniqueLowerNames(csr.DNSNames)
}
