package checker

import (
	"crypto/rand"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/crl/crl_x509"
	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/boulder/test"
)

func TestValidate(t *testing.T) {
	crlFile, err := os.Open("../../test/hierarchy/int-e1.crl.pem")
	test.AssertNotError(t, err, "opening test crl file")
	crlPEM, err := io.ReadAll(crlFile)
	test.AssertNotError(t, err, "reading test crl file")
	crlDER, _ := pem.Decode(crlPEM)
	crl, err := crl_x509.ParseRevocationList(crlDER.Bytes)
	test.AssertNotError(t, err, "parsing test crl")
	issuer, err := core.LoadCert("../../test/hierarchy/int-e1.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")

	err = Validate(crl, issuer, 100*365*24*time.Hour)
	test.AssertNotError(t, err, "validating good crl")

	err = Validate(crl, issuer, 0)
	test.AssertError(t, err, "validating too-old crl")
	test.AssertContains(t, err.Error(), "in the past")

	issuer2, err := core.LoadCert("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "loading test issuer")
	err = Validate(crl, issuer2, 100*365*24*time.Hour)
	test.AssertError(t, err, "validating crl from wrong issuer")
	test.AssertContains(t, err.Error(), "signature")

	crl.Number = nil
	err = Validate(crl, issuer, 100*365*24*time.Hour)
	test.AssertError(t, err, "validaint crl with lint error")
	test.AssertContains(t, err.Error(), "linting")
}

func TestDiff(t *testing.T) {
	issuer, signer, err := issuance.LoadIssuer(issuance.IssuerLoc{
		File:     "../../test/hierarchy/int-e1.key.pem",
		CertFile: "../../test/hierarchy/int-e1.cert.pem",
	})
	test.AssertNotError(t, err, "loading test issuer")

	now := time.Now()
	template := crl_x509.RevocationList{
		ThisUpdate: now,
		NextUpdate: now.Add(24 * time.Hour),
		Number:     big.NewInt(1),
		RevokedCertificates: []crl_x509.RevokedCertificate{
			{
				SerialNumber:   big.NewInt(1),
				RevocationTime: now.Add(-time.Hour),
			},
			{
				SerialNumber:   big.NewInt(2),
				RevocationTime: now.Add(-time.Hour),
			},
		},
	}

	oldCRLDER, err := crl_x509.CreateRevocationList(rand.Reader, &template, issuer.Certificate, signer)
	test.AssertNotError(t, err, "creating old crl")
	oldCRL, err := crl_x509.ParseRevocationList(oldCRLDER)
	test.AssertNotError(t, err, "parsing old crl")

	now = now.Add(time.Hour)
	template = crl_x509.RevocationList{
		ThisUpdate: now,
		NextUpdate: now.Add(24 * time.Hour),
		Number:     big.NewInt(2),
		RevokedCertificates: []crl_x509.RevokedCertificate{
			{
				SerialNumber:   big.NewInt(1),
				RevocationTime: now.Add(-2 * time.Hour),
			},
			{
				SerialNumber:   big.NewInt(3),
				RevocationTime: now.Add(-time.Hour),
			},
		},
	}

	newCRLDER, err := crl_x509.CreateRevocationList(rand.Reader, &template, issuer.Certificate, signer)
	test.AssertNotError(t, err, "creating old crl")
	newCRL, err := crl_x509.ParseRevocationList(newCRLDER)
	test.AssertNotError(t, err, "parsing old crl")

	res, err := Diff(oldCRL, newCRL)
	test.AssertNotError(t, err, "diffing crls")
	test.AssertEquals(t, len(res.Added), 1)
	test.AssertEquals(t, len(res.Removed), 1)
}
