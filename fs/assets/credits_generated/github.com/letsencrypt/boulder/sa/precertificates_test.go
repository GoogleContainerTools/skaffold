package sa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
)

// findIssuedName is a small helper test function to directly query the
// issuedNames table for a given name to find a serial (or return an err).
func findIssuedName(dbMap db.OneSelector, name string) (string, error) {
	var issuedNamesSerial string
	err := dbMap.SelectOne(
		&issuedNamesSerial,
		`SELECT serial FROM issuedNames
		WHERE reversedName = ?
		ORDER BY notBefore DESC
		LIMIT 1`,
		ReverseName(name))
	return issuedNamesSerial, err
}

func TestAddSerial(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)
	serial, testCert := test.ThrowAwayCert(t, 1)

	_, err := sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		RegID:   reg.Id,
		Created: testCert.NotBefore.UnixNano(),
		Expires: testCert.NotAfter.UnixNano(),
	})
	test.AssertError(t, err, "adding without serial should fail")

	_, err = sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		Serial:  serial,
		Created: testCert.NotBefore.UnixNano(),
		Expires: testCert.NotAfter.UnixNano(),
	})
	test.AssertError(t, err, "adding without regid should fail")

	_, err = sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		Serial:  serial,
		RegID:   reg.Id,
		Expires: testCert.NotAfter.UnixNano(),
	})
	test.AssertError(t, err, "adding without created should fail")

	_, err = sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		Serial:  serial,
		RegID:   reg.Id,
		Created: testCert.NotBefore.UnixNano(),
	})
	test.AssertError(t, err, "adding without expires should fail")

	_, err = sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		Serial:  serial,
		RegID:   reg.Id,
		Created: testCert.NotBefore.UnixNano(),
		Expires: testCert.NotAfter.UnixNano(),
	})
	test.AssertNotError(t, err, "adding serial should have succeeded")
}

func TestGetSerialMetadata(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)
	serial, _ := test.ThrowAwayCert(t, 1)

	_, err := sa.GetSerialMetadata(context.Background(), &sapb.Serial{Serial: serial})
	test.AssertError(t, err, "getting nonexistent serial should have failed")

	_, err = sa.AddSerial(context.Background(), &sapb.AddSerialRequest{
		Serial:  serial,
		RegID:   reg.Id,
		Created: clk.Now().UnixNano(),
		Expires: clk.Now().Add(time.Hour).UnixNano(),
	})
	test.AssertNotError(t, err, "failed to add test serial")

	m, err := sa.GetSerialMetadata(context.Background(), &sapb.Serial{Serial: serial})

	test.AssertNotError(t, err, "getting serial should have succeeded")
	test.AssertEquals(t, m.Serial, serial)
	test.AssertEquals(t, m.RegistrationID, reg.Id)
	test.AssertEquals(t, time.Unix(0, m.Created).UTC(), clk.Now())
	test.AssertEquals(t, time.Unix(0, m.Expires).UTC(), clk.Now().Add(time.Hour))
}

func TestAddPrecertificate(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	// Create a throw-away self signed certificate with a random name and
	// serial number
	serial, testCert := test.ThrowAwayCert(t, 1)

	// Add the cert as a precertificate
	ocspResp := []byte{0, 0, 1}
	regID := reg.Id
	issuedTime := time.Date(2018, 4, 1, 7, 0, 0, 0, time.UTC)
	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		RegID:    regID,
		Ocsp:     ocspResp,
		Issued:   issuedTime.UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test cert")

	// It should have the expected certificate status
	certStatus, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "Couldn't get status for test cert")
	test.Assert(
		t,
		bytes.Equal(certStatus.OcspResponse, ocspResp),
		fmt.Sprintf("OCSP responses don't match, expected: %x, got %x", certStatus.OcspResponse, ocspResp),
	)
	test.AssertEquals(t, clk.Now().UnixNano(), certStatus.OcspLastUpdated)

	// It should show up in the issued names table
	issuedNamesSerial, err := findIssuedName(sa.dbMap, testCert.DNSNames[0])
	test.AssertNotError(t, err, "expected no err querying issuedNames for precert")
	test.AssertEquals(t, issuedNamesSerial, serial)

	// We should also be able to call AddCertificate with the same cert
	// without it being an error. The duplicate err on inserting to
	// issuedNames should be ignored.
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    testCert.Raw,
		RegID:  regID,
		Issued: issuedTime.UnixNano(),
	})
	test.AssertNotError(t, err, "unexpected err adding final cert after precert")
}

func TestAddPrecertificateNoOCSP(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)
	_, testCert := test.ThrowAwayCert(t, 1)

	regID := reg.Id
	issuedTime := time.Date(2018, 4, 1, 7, 0, 0, 0, time.UTC)
	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		RegID:    regID,
		Issued:   issuedTime.UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test cert")
}

func TestAddPreCertificateDuplicate(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	_, testCert := test.ThrowAwayCert(t, 1)

	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		Issued:   clk.Now().UnixNano(),
		RegID:    reg.Id,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add test certificate")

	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		Issued:   clk.Now().UnixNano(),
		RegID:    reg.Id,
		IssuerID: 1,
	})
	test.AssertDeepEquals(t, err, berrors.DuplicateError("cannot add a duplicate cert"))
}

func TestAddPrecertificateIncomplete(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	// Create a throw-away self signed certificate with a random name and
	// serial number
	_, testCert := test.ThrowAwayCert(t, 1)

	// Add the cert as a precertificate
	ocspResp := []byte{0, 0, 1}
	regID := reg.Id
	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:    testCert.Raw,
		RegID:  regID,
		Ocsp:   ocspResp,
		Issued: time.Date(2018, 4, 1, 7, 0, 0, 0, time.UTC).UnixNano(),
		// Leaving out IssuerID
	})

	test.AssertError(t, err, "Adding precert with no issuer did not fail")
}

func TestAddPrecertificateKeyHash(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()
	reg := createWorkingRegistration(t, sa)

	serial, testCert := test.ThrowAwayCert(t, 1)
	_, err := sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      testCert.Raw,
		RegID:    reg.Id,
		Ocsp:     []byte{1, 2, 3},
		Issued:   testCert.NotBefore.UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "failed to add precert")

	var keyHashes []keyHashModel
	_, err = sa.dbMap.Select(&keyHashes, "SELECT * FROM keyHashToSerial")
	test.AssertNotError(t, err, "failed to retrieve rows from keyHashToSerial")
	test.AssertEquals(t, len(keyHashes), 1)
	test.AssertEquals(t, keyHashes[0].CertSerial, serial)
	test.AssertEquals(t, keyHashes[0].CertNotAfter, testCert.NotAfter)
	spkiHash := sha256.Sum256(testCert.RawSubjectPublicKeyInfo)
	test.Assert(t, bytes.Equal(keyHashes[0].KeyHash, spkiHash[:]), "spki hash mismatch")
}
