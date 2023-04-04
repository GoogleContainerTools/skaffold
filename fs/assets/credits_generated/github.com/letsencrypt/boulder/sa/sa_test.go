package sa

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	mrand "math/rand"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/probs"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
	"github.com/letsencrypt/boulder/test/vars"
	"golang.org/x/crypto/ocsp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	jose "gopkg.in/go-jose/go-jose.v2"
)

var log = blog.UseMock()
var ctx = context.Background()

var (
	theKey = `{
    "kty": "RSA",
    "n": "n4EPtAOCc9AlkeQHPzHStgAbgs7bTZLwUBZdR8_KuKPEHLd4rHVTeT-O-XV2jRojdNhxJWTDvNd7nqQ0VEiZQHz_AJmSCpMaJMRBSFKrKb2wqVwGU_NsYOYL-QtiWN2lbzcEe6XC0dApr5ydQLrHqkHHig3RBordaZ6Aj-oBHqFEHYpPe7Tpe-OfVfHd1E6cS6M1FZcD1NNLYD5lFHpPI9bTwJlsde3uhGqC0ZCuEHg8lhzwOHrtIQbS0FVbb9k3-tVTU4fg_3L_vniUFAKwuCLqKnS2BYwdq_mzSnbLY7h_qixoR7jig3__kRhuaxwUkRz5iaiQkqgc5gHdrNP5zw",
    "e": "AQAB"
}`
)

// initSA constructs a SQLStorageAuthority and a clean up function
// that should be defer'ed to the end of the test.
func initSA(t *testing.T) (*SQLStorageAuthority, clock.FakeClock, func()) {
	t.Helper()
	features.Reset()

	dbMap, err := NewDbMap(vars.DBConnSA, DbSettings{})
	if err != nil {
		t.Fatalf("Failed to create dbMap: %s", err)
	}

	dbIncidentsMap, err := NewDbMap(vars.DBConnIncidents, DbSettings{})
	if err != nil {
		t.Fatalf("Failed to create dbMap: %s", err)
	}

	fc := clock.NewFake()
	fc.Set(time.Date(2015, 3, 4, 5, 0, 0, 0, time.UTC))

	saro, err := NewSQLStorageAuthorityRO(dbMap, dbIncidentsMap, 1, fc, log)
	if err != nil {
		t.Fatalf("Failed to create SA: %s", err)
	}

	sa, err := NewSQLStorageAuthorityWrapping(saro, dbMap, metrics.NoopRegisterer)
	if err != nil {
		t.Fatalf("Failed to create SA: %s", err)
	}

	return sa, fc, test.ResetBoulderTestDatabase(t)
}

// CreateWorkingTestRegistration inserts a new, correct Registration into the
// given SA.
func createWorkingRegistration(t *testing.T, sa *SQLStorageAuthority) *corepb.Registration {
	initialIP, _ := net.ParseIP("88.77.66.11").MarshalText()
	reg, err := sa.NewRegistration(context.Background(), &corepb.Registration{
		Key:       []byte(theKey),
		Contact:   []string{"mailto:foo@example.com"},
		InitialIP: initialIP,
		CreatedAt: time.Date(2003, 5, 10, 0, 0, 0, 0, time.UTC).UnixNano(),
		Status:    string(core.StatusValid),
	})
	if err != nil {
		t.Fatalf("Unable to create new registration: %s", err)
	}
	return reg
}

func createPendingAuthorization(t *testing.T, sa *SQLStorageAuthority, domain string, exp time.Time) int64 {
	t.Helper()

	tokenStr := core.NewToken()
	token, err := base64.RawURLEncoding.DecodeString(tokenStr)
	test.AssertNotError(t, err, "computing test authorization challenge token")

	am := authzModel{
		IdentifierType:  0, // dnsName
		IdentifierValue: domain,
		RegistrationID:  1,
		Status:          statusToUint[core.StatusPending],
		Expires:         exp,
		Challenges:      1 << challTypeToUint[string(core.ChallengeTypeHTTP01)],
		Token:           token,
	}

	err = sa.dbMap.Insert(&am)
	test.AssertNotError(t, err, "creating test authorization")

	return am.ID
}

func createFinalizedAuthorization(t *testing.T, sa *SQLStorageAuthority, domain string, exp time.Time,
	status string, attemptedAt time.Time) int64 {
	t.Helper()
	pendingID := createPendingAuthorization(t, sa, domain, exp)
	expInt := exp.UnixNano()
	attempted := string(core.ChallengeTypeHTTP01)
	attemptedAtInt := attemptedAt.UnixNano()
	_, err := sa.FinalizeAuthorization2(context.Background(), &sapb.FinalizeAuthorizationRequest{
		Id:          pendingID,
		Status:      status,
		Expires:     expInt,
		Attempted:   attempted,
		AttemptedAt: attemptedAtInt,
	})
	test.AssertNotError(t, err, "sa.FinalizeAuthorizations2 failed")
	return pendingID
}

func goodTestJWK() *jose.JSONWebKey {
	var jwk jose.JSONWebKey
	err := json.Unmarshal([]byte(theKey), &jwk)
	if err != nil {
		panic("known-good theKey is no longer known-good")
	}
	return &jwk
}

func TestAddRegistration(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	jwk := goodTestJWK()
	jwkJSON, _ := jwk.MarshalJSON()

	contacts := []string{"mailto:foo@example.com"}
	initialIP, _ := net.ParseIP("43.34.43.34").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       jwkJSON,
		Contact:   contacts,
		InitialIP: initialIP,
	})
	if err != nil {
		t.Fatalf("Couldn't create new registration: %s", err)
	}
	test.Assert(t, reg.Id != 0, "ID shouldn't be 0")
	test.AssertDeepEquals(t, reg.Contact, contacts)

	_, err = sa.GetRegistration(ctx, &sapb.RegistrationID{Id: 0})
	test.AssertError(t, err, "Registration object for ID 0 was returned")

	dbReg, err := sa.GetRegistration(ctx, &sapb.RegistrationID{Id: reg.Id})
	test.AssertNotError(t, err, fmt.Sprintf("Couldn't get registration with ID %v", reg.Id))

	createdAt := clk.Now()
	test.AssertEquals(t, dbReg.Id, reg.Id)
	test.AssertByteEquals(t, dbReg.Key, jwkJSON)
	test.AssertDeepEquals(t, dbReg.CreatedAt, createdAt.UnixNano())

	initialIP, _ = net.ParseIP("72.72.72.72").MarshalText()
	newReg := &corepb.Registration{
		Id:        reg.Id,
		Key:       jwkJSON,
		Contact:   []string{"test.com"},
		InitialIP: initialIP,
		Agreement: "yes",
	}
	_, err = sa.UpdateRegistration(ctx, newReg)
	test.AssertNotError(t, err, fmt.Sprintf("Couldn't get registration with ID %v", reg.Id))
	dbReg, err = sa.GetRegistrationByKey(ctx, &sapb.JSONWebKey{Jwk: jwkJSON})
	test.AssertNotError(t, err, "Couldn't get registration by key")

	test.AssertEquals(t, dbReg.Id, newReg.Id)
	test.AssertEquals(t, dbReg.Agreement, newReg.Agreement)

	anotherKey := `{
		"kty":"RSA",
		"n": "vd7rZIoTLEe-z1_8G1FcXSw9CQFEJgV4g9V277sER7yx5Qjz_Pkf2YVth6wwwFJEmzc0hoKY-MMYFNwBE4hQHw",
		"e":"AQAB"
	}`

	_, err = sa.GetRegistrationByKey(ctx, &sapb.JSONWebKey{Jwk: []byte(anotherKey)})
	test.AssertError(t, err, "Registration object for invalid key was returned")
}

func TestNoSuchRegistrationErrors(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	_, err := sa.GetRegistration(ctx, &sapb.RegistrationID{Id: 100})
	test.AssertErrorIs(t, err, berrors.NotFound)

	jwk := goodTestJWK()
	jwkJSON, _ := jwk.MarshalJSON()

	_, err = sa.GetRegistrationByKey(ctx, &sapb.JSONWebKey{Jwk: jwkJSON})
	test.AssertErrorIs(t, err, berrors.NotFound)

	_, err = sa.UpdateRegistration(ctx, &corepb.Registration{Id: 100, Key: jwkJSON, InitialIP: []byte("foo")})
	test.AssertErrorIs(t, err, berrors.NotFound)
}

func TestAddCertificate(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	// An example cert taken from EFF's website
	certDER, err := os.ReadFile("www.eff.org.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")

	// Calling AddCertificate with a non-nil issued should succeed
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  reg.Id,
		Issued: sa.clk.Now().UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add www.eff.org.der")

	retrievedCert, err := sa.GetCertificate(ctx, &sapb.Serial{Serial: "000000000000000000000000000000021bd4"})
	test.AssertNotError(t, err, "Couldn't get www.eff.org.der by full serial")
	test.AssertByteEquals(t, certDER, retrievedCert.Der)
	// Because nil was provided as the Issued time we expect the cert was stored
	// with an issued time equal to now
	test.AssertEquals(t, retrievedCert.Issued, clk.Now().UnixNano())

	// Test cert generated locally by Boulder, with names [example.com,
	// www.example.com, admin.example.com]
	certDER2, err := os.ReadFile("test-cert.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")
	serial := "ffdd9b8a82126d96f61d378d5ba99a0474f0"

	// Add the certificate with a specific issued time instead of nil
	issuedTime := time.Date(2018, 4, 1, 7, 0, 0, 0, time.UTC)
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER2,
		RegID:  reg.Id,
		Issued: issuedTime.UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.der")

	retrievedCert2, err := sa.GetCertificate(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "Couldn't get test-cert.der")
	test.AssertByteEquals(t, certDER2, retrievedCert2.Der)
	// The cert should have been added with the specific issued time we provided
	// as the issued field.
	test.AssertEquals(t, retrievedCert2.Issued, issuedTime.UnixNano())

	// Test adding OCSP response with cert
	certDER3, err := os.ReadFile("test-cert2.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")
	ocspResp := []byte{0, 0, 1}
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER3,
		RegID:  reg.Id,
		Ocsp:   ocspResp,
		Issued: issuedTime.UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add test-cert2.der")
}

func TestAddCertificateDuplicate(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	_, testCert := test.ThrowAwayCert(t, 1)

	issuedTime := clk.Now()
	_, err := sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    testCert.Raw,
		RegID:  reg.Id,
		Issued: issuedTime.UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add test certificate")

	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    testCert.Raw,
		RegID:  reg.Id,
		Issued: issuedTime.UnixNano(),
	})
	test.AssertDeepEquals(t, err, berrors.DuplicateError("cannot add a duplicate cert"))

}

func TestCountCertificatesByNames(t *testing.T) {
	sa, clk, cleanUp := initSA(t)
	defer cleanUp()

	// Test cert generated locally by Boulder, with names [example.com,
	// www.example.com, admin.example.com]
	certDER, err := os.ReadFile("test-cert.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")

	cert, err := x509.ParseCertificate(certDER)
	test.AssertNotError(t, err, "Couldn't parse example cert DER")

	// Set the test clock's time to the time from the test certificate, plus an
	// hour to account for rounding.
	clk.Add(time.Hour - clk.Now().Sub(cert.NotBefore))
	now := clk.Now()
	yesterday := clk.Now().Add(-24 * time.Hour).UnixNano()
	twoDaysAgo := clk.Now().Add(-48 * time.Hour).UnixNano()
	tomorrow := clk.Now().Add(24 * time.Hour).UnixNano()

	// Count for a name that doesn't have any certs
	req := &sapb.CountCertificatesByNamesRequest{
		Names: []string{"example.com"},
		Range: &sapb.Range{
			Earliest: yesterday,
			Latest:   now.UnixNano(),
		},
	}
	counts, err := sa.CountCertificatesByNames(ctx, req)
	test.AssertNotError(t, err, "Error counting certs.")
	test.AssertEquals(t, len(counts.Counts), 1)
	test.AssertEquals(t, counts.Counts["example.com"], int64(0))

	// Add the test cert and query for its names.
	reg := createWorkingRegistration(t, sa)
	issued := sa.clk.Now()
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add test-cert.der")

	// Time range including now should find the cert
	counts, err = sa.CountCertificatesByNames(ctx, req)
	test.AssertNotError(t, err, "sa.CountCertificatesByName failed")
	test.AssertEquals(t, len(counts.Counts), 1)
	test.AssertEquals(t, counts.Counts["example.com"], int64(1))

	// Time range between two days ago and yesterday should not.
	req.Range.Earliest = twoDaysAgo
	req.Range.Latest = yesterday
	counts, err = sa.CountCertificatesByNames(ctx, req)
	test.AssertNotError(t, err, "Error counting certs.")
	test.AssertEquals(t, len(counts.Counts), 1)
	test.AssertEquals(t, counts.Counts["example.com"], int64(0))

	// Time range between now and tomorrow also should not (time ranges are
	// inclusive at the tail end, but not the beginning end).
	req.Range.Earliest = now.UnixNano()
	req.Range.Latest = tomorrow
	counts, err = sa.CountCertificatesByNames(ctx, req)
	test.AssertNotError(t, err, "Error counting certs.")
	test.AssertEquals(t, len(counts.Counts), 1)
	test.AssertEquals(t, counts.Counts["example.com"], int64(0))

	// Add a second test cert (for example.co.bn) and query for multiple names.
	names := []string{"example.com", "foo.com", "example.co.bn"}

	// Override countCertificatesByName with an implementation of certCountFunc
	// that will block forever if it's called in serial, but will succeed if
	// called in parallel.
	var interlocker sync.WaitGroup
	interlocker.Add(len(names))
	sa.parallelismPerRPC = len(names)
	oldCertCountFunc := sa.countCertificatesByName
	sa.countCertificatesByName = func(sel db.Selector, domain string, timeRange *sapb.Range) (int64, time.Time, error) {
		interlocker.Done()
		interlocker.Wait()
		return oldCertCountFunc(sel, domain, timeRange)
	}

	certDER2, err := os.ReadFile("test-cert2.der")
	test.AssertNotError(t, err, "Couldn't read test-cert2.der")
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER2,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Couldn't add test-cert2.der")
	req.Names = names
	req.Range.Earliest = yesterday
	req.Range.Latest = now.Add(10000 * time.Hour).UnixNano()
	counts, err = sa.CountCertificatesByNames(ctx, req)
	test.AssertNotError(t, err, "Error counting certs.")
	test.AssertEquals(t, len(counts.Counts), 3)

	expected := map[string]int64{
		"example.co.bn": 1,
		"foo.com":       0,
		"example.com":   1,
	}
	for name, count := range counts.Counts {
		domain := name
		actualCount := count
		expectedCount := expected[domain]
		test.AssertEquals(t, actualCount, expectedCount)
	}
}

func TestCountRegistrationsByIP(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	contact := []string{"mailto:foo@example.com"}

	// Create one IPv4 registration
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("43.34.43.34").MarshalText()
	_, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	// Create two IPv6 registrations, both within the same /48
	key, _ = jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(2), E: 1}}.MarshalJSON()
	initialIP, _ = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9652").MarshalText()
	test.AssertNotError(t, err, "Couldn't insert registration")
	_, err = sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	test.AssertNotError(t, err, "Couldn't insert registration")
	key, _ = jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(3), E: 1}}.MarshalJSON()
	initialIP, _ = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9653").MarshalText()
	_, err = sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	test.AssertNotError(t, err, "Couldn't insert registration")

	req := &sapb.CountRegistrationsByIPRequest{
		Ip: net.ParseIP("1.1.1.1"),
		Range: &sapb.Range{
			Earliest: fc.Now().Add(-time.Hour * 24).UnixNano(),
			Latest:   fc.Now().UnixNano(),
		},
	}

	// There should be 0 registrations for an IPv4 address we didn't add
	// a registration for
	count, err := sa.CountRegistrationsByIP(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(0))
	// There should be 1 registration for the IPv4 address we did add
	// a registration for.
	req.Ip = net.ParseIP("43.34.43.34")
	count, err = sa.CountRegistrationsByIP(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(1))
	// There should be 1 registration for the first IPv6 address we added
	// a registration for
	req.Ip = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9652")
	count, err = sa.CountRegistrationsByIP(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(1))
	// There should be 1 registration for the second IPv6 address we added
	// a registration for as well
	req.Ip = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9653")
	count, err = sa.CountRegistrationsByIP(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(1))
	// There should be 0 registrations for an IPv6 address in the same /48 as the
	// two IPv6 addresses with registrations
	req.Ip = net.ParseIP("2001:cdba:1234:0000:0000:0000:0000:0000")
	count, err = sa.CountRegistrationsByIP(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(0))
}

func TestCountRegistrationsByIPRange(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	contact := []string{"mailto:foo@example.com"}

	// Create one IPv4 registration
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("43.34.43.34").MarshalText()
	_, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	// Create two IPv6 registrations, both within the same /48
	key, _ = jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(2), E: 1}}.MarshalJSON()
	initialIP, _ = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9652").MarshalText()
	test.AssertNotError(t, err, "Couldn't insert registration")
	_, err = sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	test.AssertNotError(t, err, "Couldn't insert registration")
	key, _ = jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(3), E: 1}}.MarshalJSON()
	initialIP, _ = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9653").MarshalText()
	_, err = sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
		Contact:   contact,
	})
	test.AssertNotError(t, err, "Couldn't insert registration")

	req := &sapb.CountRegistrationsByIPRequest{
		Ip: net.ParseIP("1.1.1.1"),
		Range: &sapb.Range{
			Earliest: fc.Now().Add(-time.Hour * 24).UnixNano(),
			Latest:   fc.Now().UnixNano(),
		},
	}

	// There should be 0 registrations in the range for an IPv4 address we didn't
	// add a registration for
	req.Ip = net.ParseIP("1.1.1.1")
	count, err := sa.CountRegistrationsByIPRange(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(0))
	// There should be 1 registration in the range for the IPv4 address we did
	// add a registration for
	req.Ip = net.ParseIP("43.34.43.34")
	count, err = sa.CountRegistrationsByIPRange(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(1))
	// There should be 2 registrations in the range for the first IPv6 address we added
	// a registration for because it's in the same /48
	req.Ip = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9652")
	count, err = sa.CountRegistrationsByIPRange(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(2))
	// There should be 2 registrations in the range for the second IPv6 address
	// we added a registration for as well, because it too is in the same /48
	req.Ip = net.ParseIP("2001:cdba:1234:5678:9101:1121:3257:9653")
	count, err = sa.CountRegistrationsByIPRange(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(2))
	// There should also be 2 registrations in the range for an arbitrary IPv6 address in
	// the same /48 as the registrations we added
	req.Ip = net.ParseIP("2001:cdba:1234:0000:0000:0000:0000:0000")
	count, err = sa.CountRegistrationsByIPRange(ctx, req)
	test.AssertNotError(t, err, "Failed to count registrations")
	test.AssertEquals(t, count.Count, int64(2))
}

func TestFQDNSets(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	tx, err := sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	names := []string{"a.example.com", "B.example.com"}
	expires := fc.Now().Add(time.Hour * 2).UTC()
	issued := fc.Now()
	err = addFQDNSet(tx, names, "serial", issued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	threeHours := time.Hour * 3
	req := &sapb.CountFQDNSetsRequest{
		Domains: names,
		Window:  threeHours.Nanoseconds(),
	}
	// only one valid
	count, err := sa.CountFQDNSets(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, count.Count, int64(1))

	// check hash isn't affected by changing name order/casing
	req.Domains = []string{"b.example.com", "A.example.COM"}
	count, err = sa.CountFQDNSets(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, count.Count, int64(1))

	// add another valid set
	tx, err = sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	err = addFQDNSet(tx, names, "anotherSerial", issued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// only two valid
	req.Domains = names
	count, err = sa.CountFQDNSets(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, count.Count, int64(2))

	// add an expired set
	tx, err = sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	err = addFQDNSet(
		tx,
		names,
		"yetAnotherSerial",
		issued.Add(-threeHours),
		expires.Add(-threeHours),
	)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// only two valid
	count, err = sa.CountFQDNSets(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, count.Count, int64(2))
}

func TestFQDNSetTimestampsForWindow(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	tx, err := sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")

	names := []string{"a.example.com", "B.example.com"}
	window := time.Hour * 3
	req := &sapb.CountFQDNSetsRequest{
		Domains: names,
		Window:  window.Nanoseconds(),
	}

	// Ensure zero issuance has occurred for names.
	resp, err := sa.FQDNSetTimestampsForWindow(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, len(resp.Timestamps), 0)

	// Add an issuance for names inside the window.
	expires := fc.Now().Add(time.Hour * 2).UTC()
	firstIssued := fc.Now()
	err = addFQDNSet(tx, names, "serial", firstIssued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// Ensure there's 1 issuance timestamp for names inside the window.
	resp, err = sa.FQDNSetTimestampsForWindow(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, len(resp.Timestamps), 1)
	test.AssertEquals(t, firstIssued, time.Unix(0, resp.Timestamps[len(resp.Timestamps)-1]).UTC())

	// Ensure that the hash isn't affected by changing name order/casing.
	req.Domains = []string{"b.example.com", "A.example.COM"}
	resp, err = sa.FQDNSetTimestampsForWindow(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, len(resp.Timestamps), 1)
	test.AssertEquals(t, firstIssued, time.Unix(0, resp.Timestamps[len(resp.Timestamps)-1]).UTC())

	// Add another issuance for names inside the window.
	tx, err = sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	err = addFQDNSet(tx, names, "anotherSerial", firstIssued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// Ensure there are two issuance timestamps for names inside the window.
	req.Domains = names
	resp, err = sa.FQDNSetTimestampsForWindow(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, len(resp.Timestamps), 2)
	test.AssertEquals(t, firstIssued, time.Unix(0, resp.Timestamps[len(resp.Timestamps)-1]).UTC())

	// Add another issuance for names but just outside the window.
	tx, err = sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	err = addFQDNSet(tx, names, "yetAnotherSerial", firstIssued.Add(-window), expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// Ensure there are still only two issuance timestamps in the window.
	resp, err = sa.FQDNSetTimestampsForWindow(ctx, req)
	test.AssertNotError(t, err, "Failed to count name sets")
	test.AssertEquals(t, len(resp.Timestamps), 2)
	test.AssertEquals(t, firstIssued, time.Unix(0, resp.Timestamps[len(resp.Timestamps)-1]).UTC())
}

func TestFQDNSetsExists(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	names := []string{"a.example.com", "B.example.com"}
	exists, err := sa.FQDNSetExists(ctx, &sapb.FQDNSetExistsRequest{Domains: names})
	test.AssertNotError(t, err, "Failed to check FQDN set existence")
	test.Assert(t, !exists.Exists, "FQDN set shouldn't exist")

	tx, err := sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	expires := fc.Now().Add(time.Hour * 2).UTC()
	issued := fc.Now()
	err = addFQDNSet(tx, names, "serial", issued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	exists, err = sa.FQDNSetExists(ctx, &sapb.FQDNSetExistsRequest{Domains: names})
	test.AssertNotError(t, err, "Failed to check FQDN set existence")
	test.Assert(t, exists.Exists, "FQDN set does exist")
}

type queryRecorder struct {
	query string
	args  []interface{}
}

func (e *queryRecorder) Query(query string, args ...interface{}) (*sql.Rows, error) {
	e.query = query
	e.args = args
	return nil, nil
}

func TestAddIssuedNames(t *testing.T) {
	serial := big.NewInt(1)
	expectedSerial := "000000000000000000000000000000000001"
	notBefore := time.Date(2018, 2, 14, 12, 0, 0, 0, time.UTC)
	placeholdersPerName := "(?,?,?,?)"
	baseQuery := "INSERT INTO issuedNames (reversedName,serial,notBefore,renewal) VALUES"

	testCases := []struct {
		Name         string
		IssuedNames  []string
		SerialNumber *big.Int
		NotBefore    time.Time
		Renewal      bool
		ExpectedArgs []interface{}
	}{
		{
			Name:         "One domain, not a renewal",
			IssuedNames:  []string{"example.co.uk"},
			SerialNumber: serial,
			NotBefore:    notBefore,
			Renewal:      false,
			ExpectedArgs: []interface{}{
				"uk.co.example",
				expectedSerial,
				notBefore,
				false,
			},
		},
		{
			Name:         "Two domains, not a renewal",
			IssuedNames:  []string{"example.co.uk", "example.xyz"},
			SerialNumber: serial,
			NotBefore:    notBefore,
			Renewal:      false,
			ExpectedArgs: []interface{}{
				"uk.co.example",
				expectedSerial,
				notBefore,
				false,
				"xyz.example",
				expectedSerial,
				notBefore,
				false,
			},
		},
		{
			Name:         "One domain, renewal",
			IssuedNames:  []string{"example.co.uk"},
			SerialNumber: serial,
			NotBefore:    notBefore,
			Renewal:      true,
			ExpectedArgs: []interface{}{
				"uk.co.example",
				expectedSerial,
				notBefore,
				true,
			},
		},
		{
			Name:         "Two domains, renewal",
			IssuedNames:  []string{"example.co.uk", "example.xyz"},
			SerialNumber: serial,
			NotBefore:    notBefore,
			Renewal:      true,
			ExpectedArgs: []interface{}{
				"uk.co.example",
				expectedSerial,
				notBefore,
				true,
				"xyz.example",
				expectedSerial,
				notBefore,
				true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var e queryRecorder
			err := addIssuedNames(
				&e,
				&x509.Certificate{
					DNSNames:     tc.IssuedNames,
					SerialNumber: tc.SerialNumber,
					NotBefore:    tc.NotBefore,
				},
				tc.Renewal)
			test.AssertNotError(t, err, "addIssuedNames failed")
			expectedPlaceholders := placeholdersPerName
			for i := 0; i < len(tc.IssuedNames)-1; i++ {
				expectedPlaceholders = fmt.Sprintf("%s,%s", expectedPlaceholders, placeholdersPerName)
			}
			expectedQuery := fmt.Sprintf("%s %s", baseQuery, expectedPlaceholders)
			test.AssertEquals(t, e.query, expectedQuery)
			if !reflect.DeepEqual(e.args, tc.ExpectedArgs) {
				t.Errorf("Wrong args: got\n%#v, expected\n%#v", e.args, tc.ExpectedArgs)
			}
		})
	}
}

func TestPreviousCertificateExists(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	// An example cert taken from EFF's website
	certDER, err := os.ReadFile("www.eff.org.der")
	test.AssertNotError(t, err, "reading cert DER")

	issued := sa.clk.Now()
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		Issued:   issued.UnixNano(),
		RegID:    reg.Id,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Failed to add precertificate")
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "calling AddCertificate")

	cases := []struct {
		name     string
		domain   string
		regID    int64
		expected bool
	}{
		{"matches", "www.eff.org", reg.Id, true},
		{"wrongDomain", "wwoof.org", reg.Id, false},
		{"wrongAccount", "www.eff.org", 3333, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			exists, err := sa.PreviousCertificateExists(context.Background(),
				&sapb.PreviousCertificateExistsRequest{
					Domain: testCase.domain,
					RegID:  testCase.regID,
				})
			test.AssertNotError(t, err, "calling PreviousCertificateExists")
			if exists.Exists != testCase.expected {
				t.Errorf("wanted %v got %v", testCase.expected, exists.Exists)
			}
		})
	}
}

func TestDeactivateAuthorization2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// deactivate a pending authorization
	expires := fc.Now().Add(time.Hour).UTC()
	attemptedAt := fc.Now()
	authzID := createPendingAuthorization(t, sa, "example.com", expires)
	_, err := sa.DeactivateAuthorization2(context.Background(), &sapb.AuthorizationID2{Id: authzID})
	test.AssertNotError(t, err, "sa.DeactivateAuthorization2 failed")

	// deactivate a valid authorization"
	authzID = createFinalizedAuthorization(t, sa, "example.com", expires, "valid", attemptedAt)
	_, err = sa.DeactivateAuthorization2(context.Background(), &sapb.AuthorizationID2{Id: authzID})
	test.AssertNotError(t, err, "sa.DeactivateAuthorization2 failed")
}

func TestDeactivateAccount(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	_, err := sa.DeactivateRegistration(context.Background(), &sapb.RegistrationID{Id: reg.Id})
	test.AssertNotError(t, err, "DeactivateRegistration failed")

	dbReg, err := sa.GetRegistration(context.Background(), &sapb.RegistrationID{Id: reg.Id})
	test.AssertNotError(t, err, "GetRegistration failed")
	test.AssertEquals(t, core.AcmeStatus(dbReg.Status), core.StatusDeactivated)
}

func TestReverseName(t *testing.T) {
	testCases := []struct {
		inputDomain   string
		inputReversed string
	}{
		{"", ""},
		{"...", "..."},
		{"com", "com"},
		{"example.com", "com.example"},
		{"www.example.com", "com.example.www"},
		{"world.wide.web.example.com", "com.example.web.wide.world"},
	}

	for _, tc := range testCases {
		output := ReverseName(tc.inputDomain)
		test.AssertEquals(t, output, tc.inputReversed)
	}
}

func TestNewOrderAndAuthzs(t *testing.T) {
	sa, _, cleanup := initSA(t)
	defer cleanup()

	// Create a test registration to reference
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	// Insert two pre-existing authorizations to reference
	idA := createPendingAuthorization(t, sa, "a.com", sa.clk.Now().Add(time.Hour))
	idB := createPendingAuthorization(t, sa, "b.com", sa.clk.Now().Add(time.Hour))
	test.AssertEquals(t, idA, int64(1))
	test.AssertEquals(t, idB, int64(2))

	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		// Insert an order for four names, two of which already have authzs
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          1,
			Names:            []string{"a.com", "b.com", "c.com", "d.com"},
			V2Authorizations: []int64{1, 2},
		},
		// And add new authorizations for the other two names.
		NewAuthzs: []*corepb.Authorization{
			{
				Identifier:     "c.com",
				RegistrationID: reg.Id,
				Expires:        sa.clk.Now().Add(time.Hour).UnixNano(),
				Status:         "pending",
				Challenges:     []*corepb.Challenge{{Token: core.NewToken()}},
			},
			{
				Identifier:     "d.com",
				RegistrationID: reg.Id,
				Expires:        sa.clk.Now().Add(time.Hour).UnixNano(),
				Status:         "pending",
				Challenges:     []*corepb.Challenge{{Token: core.NewToken()}},
			},
		},
	})
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")
	test.AssertEquals(t, order.Id, int64(1))
	test.AssertDeepEquals(t, order.V2Authorizations, []int64{1, 2, 3, 4})

	var authzIDs []int64
	_, err = sa.dbMap.Select(&authzIDs, "SELECT authzID FROM orderToAuthz2 WHERE orderID = ?;", order.Id)
	test.AssertNotError(t, err, "Failed to count orderToAuthz entries")
	test.AssertEquals(t, len(authzIDs), 4)
	test.AssertDeepEquals(t, authzIDs, []int64{1, 2, 3, 4})

	names, err := namesForOrder(sa.dbReadOnlyMap, order.Id)
	test.AssertNotError(t, err, "namesForOrder errored")
	test.AssertEquals(t, len(names), 4)
	test.AssertDeepEquals(t, names, []string{"com.a", "com.b", "com.c", "com.d"})
}

// TestNewOrderAndAuthzs_NonNilInnerOrder verifies that a nil
// sapb.NewOrderAndAuthzsRequest NewOrder object returns an error.
func TestNewOrderAndAuthzs_NonNilInnerOrder(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("17.17.17.17").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	_, err = sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewAuthzs: []*corepb.Authorization{
			{
				Identifier:     "a.com",
				RegistrationID: reg.Id,
				Expires:        fc.Now().Add(2 * time.Hour).UnixNano(),
				Status:         "pending",
				Challenges:     []*corepb.Challenge{{Token: core.NewToken()}},
			},
		},
	})
	test.AssertErrorIs(t, err, errIncompleteRequest)
}

func TestNewOrderAndAuthzs_NewAuthzExpectedFields(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	// Create a test registration to reference.
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("17.17.17.17").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	expires := fc.Now().Add(time.Hour).UnixNano()
	domain := "a.com"

	// Create an authz that does not yet exist in the database with some invalid
	// data smuggled in.
	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewAuthzs: []*corepb.Authorization{
			{
				Identifier:     domain,
				RegistrationID: reg.Id,
				Expires:        expires,
				Status:         string(core.StatusPending),
				Challenges: []*corepb.Challenge{
					{
						Status: "real fake garbage data",
						Token:  core.NewToken(),
					},
				},
			},
		},
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID: reg.Id,
			Expires:        expires,
			Names:          []string{domain},
		},
	})
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")

	// Safely get the authz for the order we created above.
	obj, err := sa.dbReadOnlyMap.Get(authzModel{}, order.V2Authorizations[0])
	test.AssertNotError(t, err, fmt.Sprintf("authorization %d not found", order.V2Authorizations[0]))

	// To access the data stored in obj at compile time, we type assert obj
	// into a pointer to an authzModel.
	am, ok := obj.(*authzModel)
	test.Assert(t, ok, "Could not type assert obj into authzModel")

	// If we're making a brand new authz, it should have the pending status
	// regardless of what incorrect status value was passed in during construction.
	test.AssertEquals(t, am.Status, statusUint(core.StatusPending))

	// Testing for the existence of these boxed nils is a definite break from
	// our paradigm of avoiding passing around boxed nils whenever possible.
	// However, the existence of these boxed nils in relation to this test is
	// actually expected. If these tests fail, then a possible SA refactor or RA
	// bug placed incorrect data into brand new authz input fields.
	test.AssertBoxedNil(t, am.Attempted, "am.Attempted should be nil")
	test.AssertBoxedNil(t, am.AttemptedAt, "am.AttemptedAt should be nil")
	test.AssertBoxedNil(t, am.ValidationError, "am.ValidationError should be nil")
	test.AssertBoxedNil(t, am.ValidationRecord, "am.ValidationRecord should be nil")
}

func TestSetOrderProcessing(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	// Create a test registration to reference
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	// Add one valid authz
	expires := fc.Now().Add(time.Hour)
	attemptedAt := fc.Now()
	authzID := createFinalizedAuthorization(t, sa, "example.com", expires, "valid", attemptedAt)

	// Add a new order in pending status with no certificate serial
	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          sa.clk.Now().Add(365 * 24 * time.Hour).UnixNano(),
			Names:            []string{"example.com"},
			V2Authorizations: []int64{authzID},
		},
	})
	test.AssertNotError(t, err, "NewOrderAndAuthzs failed")

	// Set the order to be processing
	_, err = sa.SetOrderProcessing(context.Background(), &sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "SetOrderProcessing failed")

	// Read the order by ID from the DB to check the status was correctly updated
	// to processing
	updatedOrder, err := sa.GetOrder(
		context.Background(),
		&sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "GetOrder failed")
	test.AssertEquals(t, updatedOrder.Status, string(core.StatusProcessing))
	test.AssertEquals(t, updatedOrder.BeganProcessing, true)

	// Try to set the same order to be processing again. We should get an error.
	_, err = sa.SetOrderProcessing(context.Background(), &sapb.OrderRequest{Id: order.Id})
	test.AssertError(t, err, "Set the same order processing twice. This should have been an error.")
	test.AssertErrorIs(t, err, berrors.OrderNotReady)
}

func TestFinalizeOrder(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	// Create a test registration to reference
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	// Add one valid authz
	expires := fc.Now().Add(time.Hour)
	attemptedAt := fc.Now()
	authzID := createFinalizedAuthorization(t, sa, "example.com", expires, "valid", attemptedAt)

	// Add a new order in pending status with no certificate serial
	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          sa.clk.Now().Add(365 * 24 * time.Hour).UnixNano(),
			Names:            []string{"example.com"},
			V2Authorizations: []int64{authzID},
		},
	})
	test.AssertNotError(t, err, "NewOrderAndAuthzs failed")

	// Set the order to processing so it can be finalized
	_, err = sa.SetOrderProcessing(ctx, &sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "SetOrderProcessing failed")

	// Finalize the order with a certificate serial
	order.CertificateSerial = "eat.serial.for.breakfast"
	_, err = sa.FinalizeOrder(context.Background(), &sapb.FinalizeOrderRequest{Id: order.Id, CertificateSerial: order.CertificateSerial})
	test.AssertNotError(t, err, "FinalizeOrder failed")

	// Read the order by ID from the DB to check the certificate serial and status
	// was correctly updated
	updatedOrder, err := sa.GetOrder(
		context.Background(),
		&sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "GetOrder failed")
	test.AssertEquals(t, updatedOrder.CertificateSerial, "eat.serial.for.breakfast")
	test.AssertEquals(t, updatedOrder.Status, string(core.StatusValid))
}

func TestOrder(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	// Create a test registration to reference
	key, _ := jose.JSONWebKey{Key: &rsa.PublicKey{N: big.NewInt(1), E: 1}}.MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	authzExpires := fc.Now().Add(time.Hour)
	authzID := createPendingAuthorization(t, sa, "example.com", authzExpires)

	// Set the order to expire in two hours
	expires := fc.Now().Add(2 * time.Hour).UnixNano()

	inputOrder := &corepb.Order{
		RegistrationID:   reg.Id,
		Expires:          expires,
		Names:            []string{"example.com"},
		V2Authorizations: []int64{authzID},
	}

	// Create the order
	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   inputOrder.RegistrationID,
			Expires:          inputOrder.Expires,
			Names:            inputOrder.Names,
			V2Authorizations: inputOrder.V2Authorizations,
		},
	})
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")

	// The Order from GetOrder should match the following expected order
	expectedOrder := &corepb.Order{
		// The registration ID, authorizations, expiry, and names should match the
		// input to NewOrderAndAuthzs
		RegistrationID:   inputOrder.RegistrationID,
		V2Authorizations: inputOrder.V2Authorizations,
		Names:            inputOrder.Names,
		Expires:          inputOrder.Expires,
		// The ID should have been set to 1 by the SA
		Id: 1,
		// The status should be pending
		Status: string(core.StatusPending),
		// The serial should be empty since this is a pending order
		CertificateSerial: "",
		// We should not be processing it
		BeganProcessing: false,
		// The created timestamp should have been set to the current time
		Created: sa.clk.Now().UnixNano(),
	}

	// Fetch the order by its ID and make sure it matches the expected
	storedOrder, err := sa.GetOrder(context.Background(), &sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "sa.GetOrder failed")
	test.AssertDeepEquals(t, storedOrder, expectedOrder)
}

// TestGetAuthorization2NoRows ensures that the GetAuthorization2 function returns
// the correct error when there are no results for the provided ID.
func TestGetAuthorization2NoRows(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	// An empty authz ID should result in a not found berror.
	id := int64(123)
	_, err := sa.GetAuthorization2(ctx, &sapb.AuthorizationID2{Id: id})
	test.AssertError(t, err, "Didn't get an error looking up non-existent authz ID")
	test.AssertErrorIs(t, err, berrors.NotFound)
}

func TestGetAuthorizations2(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	reg := createWorkingRegistration(t, sa)
	exp := fc.Now().AddDate(0, 0, 10).UTC()
	attemptedAt := fc.Now()

	identA := "aaa"
	identB := "bbb"
	identC := "ccc"
	identD := "ddd"
	idents := []string{identA, identB, identC}

	authzIDA := createFinalizedAuthorization(t, sa, "aaa", exp, "valid", attemptedAt)
	authzIDB := createPendingAuthorization(t, sa, "bbb", exp)
	nearbyExpires := fc.Now().UTC().Add(time.Hour)
	authzIDC := createPendingAuthorization(t, sa, "ccc", nearbyExpires)

	// Associate authorizations with an order so that GetAuthorizations2 thinks
	// they are WFE2 authorizations.
	err := sa.dbMap.Insert(&orderToAuthzModel{
		OrderID: 1,
		AuthzID: authzIDA,
	})
	test.AssertNotError(t, err, "sa.dbMap.Insert failed")
	err = sa.dbMap.Insert(&orderToAuthzModel{
		OrderID: 1,
		AuthzID: authzIDB,
	})
	test.AssertNotError(t, err, "sa.dbMap.Insert failed")
	err = sa.dbMap.Insert(&orderToAuthzModel{
		OrderID: 1,
		AuthzID: authzIDC,
	})
	test.AssertNotError(t, err, "sa.dbMap.Insert failed")

	// Set an expiry cut off of 1 day in the future similar to `RA.NewOrderAndAuthzs`. This
	// should exclude pending authorization C based on its nearbyExpires expiry
	// value.
	expiryCutoff := fc.Now().AddDate(0, 0, 1).UnixNano()
	// Get authorizations for the names used above.
	authz, err := sa.GetAuthorizations2(context.Background(), &sapb.GetAuthorizationsRequest{
		RegistrationID: reg.Id,
		Domains:        idents,
		Now:            expiryCutoff,
	})
	// It should not fail
	test.AssertNotError(t, err, "sa.GetAuthorizations2 failed")
	// We should get back two authorizations since one of the three authorizations
	// created above expires too soon.
	test.AssertEquals(t, len(authz.Authz), 2)

	// Get authorizations for the names used above, and one name that doesn't exist
	authz, err = sa.GetAuthorizations2(context.Background(), &sapb.GetAuthorizationsRequest{
		RegistrationID: reg.Id,
		Domains:        append(idents, identD),
		Now:            expiryCutoff,
	})
	// It should not fail
	test.AssertNotError(t, err, "sa.GetAuthorizations2 failed")
	// It should still return only two authorizations
	test.AssertEquals(t, len(authz.Authz), 2)
}

func TestCountOrders(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)
	now := sa.clk.Now()
	expires := now.Add(24 * time.Hour)

	req := &sapb.CountOrdersRequest{
		AccountID: 12345,
		Range: &sapb.Range{
			Earliest: now.Add(-time.Hour).UnixNano(),
			Latest:   now.Add(time.Second).UnixNano(),
		},
	}

	// Counting new orders for a reg ID that doesn't exist should return 0
	count, err := sa.CountOrders(ctx, req)
	test.AssertNotError(t, err, "Couldn't count new orders for fake reg ID")
	test.AssertEquals(t, count.Count, int64(0))

	// Add a pending authorization
	authzID := createPendingAuthorization(t, sa, "example.com", expires)

	// Add one pending order
	order, err := sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          expires.UnixNano(),
			Names:            []string{"example.com"},
			V2Authorizations: []int64{authzID},
		},
	})
	test.AssertNotError(t, err, "Couldn't create new pending order")

	// Counting new orders for the reg ID should now yield 1
	req.AccountID = reg.Id
	count, err = sa.CountOrders(ctx, req)
	test.AssertNotError(t, err, "Couldn't count new orders for reg ID")
	test.AssertEquals(t, count.Count, int64(1))

	// Moving the count window to after the order was created should return the
	// count to 0
	earliest := time.Unix(0, order.Created).Add(time.Minute)
	req.Range.Earliest = earliest.UnixNano()
	req.Range.Latest = earliest.Add(time.Hour).UnixNano()
	count, err = sa.CountOrders(ctx, req)
	test.AssertNotError(t, err, "Couldn't count new orders for reg ID")
	test.AssertEquals(t, count.Count, int64(0))
}

func TestFasterGetOrderForNames(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	domain := "example.com"
	expires := fc.Now().Add(time.Hour)

	key, _ := goodTestJWK().MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	reg, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	authzIDs := createPendingAuthorization(t, sa, domain, expires)

	expiresNano := expires.UnixNano()
	_, err = sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          expiresNano,
			V2Authorizations: []int64{authzIDs},
			Names:            []string{domain},
		},
	})
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")

	_, err = sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          expiresNano,
			V2Authorizations: []int64{authzIDs},
			Names:            []string{domain},
		},
	})
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")

	_, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: reg.Id,
		Names:  []string{domain},
	})
	test.AssertNotError(t, err, "sa.GetOrderForNames failed")
}

func TestGetOrderForNames(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// Give the order we create a short lifetime
	orderLifetime := time.Hour
	expires := fc.Now().Add(orderLifetime).UnixNano()

	// Create two test registrations to associate with orders
	key, _ := goodTestJWK().MarshalJSON()
	initialIP, _ := net.ParseIP("42.42.42.42").MarshalText()
	regA, err := sa.NewRegistration(ctx, &corepb.Registration{
		Key:       key,
		InitialIP: initialIP,
	})
	test.AssertNotError(t, err, "Couldn't create test registration")

	// Add one pending authz for the first name for regA and one
	// pending authz for the second name for regA
	authzExpires := fc.Now().Add(time.Hour)
	authzIDA := createPendingAuthorization(t, sa, "example.com", authzExpires)
	authzIDB := createPendingAuthorization(t, sa, "just.another.example.com", authzExpires)

	ctx := context.Background()
	names := []string{"example.com", "just.another.example.com"}

	// Call GetOrderForNames for a set of names we haven't created an order for
	// yet
	result, err := sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regA.Id,
		Names:  names,
	})
	// We expect the result to return an error
	test.AssertError(t, err, "sa.GetOrderForNames did not return an error for an empty result")
	// The error should be a notfound error
	test.AssertErrorIs(t, err, berrors.NotFound)
	// The result should be nil
	test.Assert(t, result == nil, "sa.GetOrderForNames for non-existent order returned non-nil result")

	// Add a new order for a set of names
	order, err := sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   regA.Id,
			Expires:          expires,
			V2Authorizations: []int64{authzIDA, authzIDB},
			Names:            names,
		},
	})
	// It shouldn't error
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")
	// The order ID shouldn't be nil
	test.AssertNotNil(t, order.Id, "NewOrderAndAuthzs returned with a nil Id")

	// Call GetOrderForNames with the same account ID and set of names as the
	// above NewOrderAndAuthzs call
	result, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regA.Id,
		Names:  names,
	})
	// It shouldn't error
	test.AssertNotError(t, err, "sa.GetOrderForNames failed")
	// The order returned should have the same ID as the order we created above
	test.AssertNotNil(t, result, "Returned order was nil")
	test.AssertEquals(t, result.Id, order.Id)

	// Call GetOrderForNames with a different account ID from the NewOrderAndAuthzs call
	regB := int64(1337)
	result, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regB,
		Names:  names,
	})
	// It should error
	test.AssertError(t, err, "sa.GetOrderForNames did not return an error for an empty result")
	// The error should be a notfound error
	test.AssertErrorIs(t, err, berrors.NotFound)
	// The result should be nil
	test.Assert(t, result == nil, "sa.GetOrderForNames for diff AcctID returned non-nil result")

	// Advance the clock beyond the initial order's lifetime
	fc.Add(2 * orderLifetime)

	// Call GetOrderForNames again with the same account ID and set of names as
	// the initial NewOrderAndAuthzs call
	result, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regA.Id,
		Names:  names,
	})
	// It should error since there is no result
	test.AssertError(t, err, "sa.GetOrderForNames did not return an error for an empty result")
	// The error should be a notfound error
	test.AssertErrorIs(t, err, berrors.NotFound)
	// The result should be nil because the initial order expired & we don't want
	// to return expired orders
	test.Assert(t, result == nil, "sa.GetOrderForNames returned non-nil result for expired order case")

	// Create two valid authorizations
	authzExpires = fc.Now().Add(time.Hour)
	attemptedAt := fc.Now()
	authzIDC := createFinalizedAuthorization(t, sa, "zombo.com", authzExpires, "valid", attemptedAt)
	authzIDD := createFinalizedAuthorization(t, sa, "welcome.to.zombo.com", authzExpires, "valid", attemptedAt)

	// Add a fresh order that uses the authorizations created above
	names = []string{"zombo.com", "welcome.to.zombo.com"}
	order, err = sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   regA.Id,
			Expires:          fc.Now().Add(orderLifetime).UnixNano(),
			V2Authorizations: []int64{authzIDC, authzIDD},
			Names:            names,
		},
	})
	// It shouldn't error
	test.AssertNotError(t, err, "sa.NewOrderAndAuthzs failed")
	// The order ID shouldn't be nil
	test.AssertNotNil(t, order.Id, "NewOrderAndAuthzs returned with a nil Id")

	// Call GetOrderForNames with the same account ID and set of names as
	// the earlier NewOrderAndAuthzs call
	result, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regA.Id,
		Names:  names,
	})
	// It should not error since a ready order can be reused.
	test.AssertNotError(t, err, "sa.GetOrderForNames returned an unexpected error for ready order reuse")
	// The order returned should have the same ID as the order we created above
	test.AssertNotNil(t, result, "sa.GetOrderForNames returned nil result")
	test.AssertEquals(t, result.Id, order.Id)

	// Set the order processing so it can be finalized
	_, err = sa.SetOrderProcessing(ctx, &sapb.OrderRequest{Id: order.Id})
	test.AssertNotError(t, err, "sa.SetOrderProcessing failed")

	// Finalize the order
	order.CertificateSerial = "cinnamon toast crunch"
	_, err = sa.FinalizeOrder(ctx, &sapb.FinalizeOrderRequest{Id: order.Id, CertificateSerial: order.CertificateSerial})
	test.AssertNotError(t, err, "sa.FinalizeOrder failed")

	// Call GetOrderForNames with the same account ID and set of names as
	// the earlier NewOrderAndAuthzs call
	result, err = sa.GetOrderForNames(ctx, &sapb.GetOrderForNamesRequest{
		AcctID: regA.Id,
		Names:  names,
	})
	// It should error since a valid order should not be reused.
	test.AssertError(t, err, "sa.GetOrderForNames did not return an error for an empty result")
	// The error should be a notfound error
	test.AssertErrorIs(t, err, berrors.NotFound)
	// The result should be nil because the one matching order has been finalized
	// already
	test.Assert(t, result == nil, "sa.GetOrderForNames returned non-nil result for finalized order case")
}

func TestStatusForOrder(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	ctx := context.Background()
	expires := fc.Now().Add(time.Hour)
	expiresNano := expires.UnixNano()
	alreadyExpired := expires.Add(-2 * time.Hour)
	attemptedAt := fc.Now()

	// Create a registration to work with
	reg := createWorkingRegistration(t, sa)

	// Create a pending authz, an expired authz, an invalid authz, a deactivated authz,
	// and a valid authz
	pendingID := createPendingAuthorization(t, sa, "pending.your.order.is.up", expires)
	expiredID := createPendingAuthorization(t, sa, "expired.your.order.is.up", alreadyExpired)
	invalidID := createFinalizedAuthorization(t, sa, "invalid.your.order.is.up", expires, "invalid", attemptedAt)
	validID := createFinalizedAuthorization(t, sa, "valid.your.order.is.up", expires, "valid", attemptedAt)
	deactivatedID := createPendingAuthorization(t, sa, "deactivated.your.order.is.up", expires)
	_, err := sa.DeactivateAuthorization2(context.Background(), &sapb.AuthorizationID2{Id: deactivatedID})
	test.AssertNotError(t, err, "sa.DeactivateAuthorization2 failed")

	testCases := []struct {
		Name             string
		AuthorizationIDs []int64
		OrderNames       []string
		OrderExpires     int64
		ExpectedStatus   string
		SetProcessing    bool
		Finalize         bool
	}{
		{
			Name:             "Order with an invalid authz",
			OrderNames:       []string{"pending.your.order.is.up", "invalid.your.order.is.up", "deactivated.your.order.is.up", "valid.your.order.is.up"},
			AuthorizationIDs: []int64{pendingID, invalidID, deactivatedID, validID},
			ExpectedStatus:   string(core.StatusInvalid),
		},
		{
			Name:             "Order with an expired authz",
			OrderNames:       []string{"pending.your.order.is.up", "expired.your.order.is.up", "deactivated.your.order.is.up", "valid.your.order.is.up"},
			AuthorizationIDs: []int64{pendingID, expiredID, deactivatedID, validID},
			ExpectedStatus:   string(core.StatusInvalid),
		},
		{
			Name:             "Order with a deactivated authz",
			OrderNames:       []string{"pending.your.order.is.up", "deactivated.your.order.is.up", "valid.your.order.is.up"},
			AuthorizationIDs: []int64{pendingID, deactivatedID, validID},
			ExpectedStatus:   string(core.StatusInvalid),
		},
		{
			Name:             "Order with a pending authz",
			OrderNames:       []string{"valid.your.order.is.up", "pending.your.order.is.up"},
			AuthorizationIDs: []int64{validID, pendingID},
			ExpectedStatus:   string(core.StatusPending),
		},
		{
			Name:             "Order with only valid authzs, not yet processed or finalized",
			OrderNames:       []string{"valid.your.order.is.up"},
			AuthorizationIDs: []int64{validID},
			ExpectedStatus:   string(core.StatusReady),
		},
		{
			Name:             "Order with only valid authzs, set processing",
			OrderNames:       []string{"valid.your.order.is.up"},
			AuthorizationIDs: []int64{validID},
			SetProcessing:    true,
			ExpectedStatus:   string(core.StatusProcessing),
		},
		{
			Name:             "Order with only valid authzs, not yet processed or finalized, OrderReadyStatus feature flag",
			OrderNames:       []string{"valid.your.order.is.up"},
			AuthorizationIDs: []int64{validID},
			ExpectedStatus:   string(core.StatusReady),
		},
		{
			Name:             "Order with only valid authzs, set processing",
			OrderNames:       []string{"valid.your.order.is.up"},
			AuthorizationIDs: []int64{validID},
			SetProcessing:    true,
			ExpectedStatus:   string(core.StatusProcessing),
		},
		{
			Name:             "Order with only valid authzs, set processing and finalized",
			OrderNames:       []string{"valid.your.order.is.up"},
			AuthorizationIDs: []int64{validID},
			SetProcessing:    true,
			Finalize:         true,
			ExpectedStatus:   string(core.StatusValid),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// If the testcase doesn't specify an order expiry use a default timestamp
			// in the near future.
			orderExpiry := tc.OrderExpires
			if orderExpiry == 0 {
				orderExpiry = expiresNano
			}
			newOrder, err := sa.NewOrderAndAuthzs(ctx, &sapb.NewOrderAndAuthzsRequest{
				NewOrder: &sapb.NewOrderRequest{
					RegistrationID:   reg.Id,
					Expires:          orderExpiry,
					V2Authorizations: tc.AuthorizationIDs,
					Names:            tc.OrderNames,
				},
			})
			test.AssertNotError(t, err, "NewOrderAndAuthzs errored unexpectedly")
			// If requested, set the order to processing
			if tc.SetProcessing {
				_, err := sa.SetOrderProcessing(ctx, &sapb.OrderRequest{Id: newOrder.Id})
				test.AssertNotError(t, err, "Error setting order to processing status")
			}
			// If requested, finalize the order
			if tc.Finalize {
				newOrder.CertificateSerial = "lucky charms"
				_, err = sa.FinalizeOrder(ctx, &sapb.FinalizeOrderRequest{Id: newOrder.Id, CertificateSerial: newOrder.CertificateSerial})
				test.AssertNotError(t, err, "Error finalizing order")
			}
			// Fetch the order by ID to get its calculated status
			storedOrder, err := sa.GetOrder(ctx, &sapb.OrderRequest{Id: newOrder.Id})
			test.AssertNotError(t, err, "GetOrder failed")
			// The status shouldn't be nil
			test.AssertNotNil(t, storedOrder.Status, "Order status was nil")
			// The status should match expected
			test.AssertEquals(t, storedOrder.Status, tc.ExpectedStatus)
		})
	}

}

func TestUpdateChallengesDeleteUnused(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	expires := fc.Now().Add(time.Hour)
	ctx := context.Background()
	attemptedAt := fc.Now()

	// Create a valid authz
	authzID := createFinalizedAuthorization(t, sa, "example.com", expires, "valid", attemptedAt)

	result, err := sa.GetAuthorization2(ctx, &sapb.AuthorizationID2{Id: authzID})
	test.AssertNotError(t, err, "sa.GetAuthorization2 failed")

	if len(result.Challenges) != 1 {
		t.Fatalf("expected 1 challenge left after finalization, got %d", len(result.Challenges))
	}
	if result.Challenges[0].Status != string(core.StatusValid) {
		t.Errorf("expected challenge status %q, got %q", core.StatusValid, result.Challenges[0].Status)
	}
	if result.Challenges[0].Type != "http-01" {
		t.Errorf("expected challenge type %q, got %q", "http-01", result.Challenges[0].Type)
	}
}

func TestRevokeCertificate(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)
	// Add a cert to the DB to test with.
	certDER, err := os.ReadFile("www.eff.org.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   sa.clk.Now().UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add www.eff.org.der")

	serial := "000000000000000000000000000000021bd4"

	status, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "GetCertificateStatus failed")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusGood)

	fc.Add(1 * time.Hour)

	now := fc.Now()
	reason := int64(1)

	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial: serial,
		Date:   now.UnixNano(),
		Reason: reason,
	})
	test.AssertError(t, err, "RevokeCertificate should fail with no response")

	response := []byte{1, 2, 3}
	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     now.UnixNano(),
		Reason:   reason,
		Response: response,
	})
	test.AssertNotError(t, err, "RevokeCertificate should have succeeded")

	status, err = sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "GetCertificateStatus failed")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusRevoked)
	test.AssertEquals(t, status.RevokedReason, reason)
	test.AssertEquals(t, status.RevokedDate, now.UnixNano())
	test.AssertEquals(t, status.OcspLastUpdated, now.UnixNano())
	test.AssertDeepEquals(t, status.OcspResponse, response)

	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     now.UnixNano(),
		Reason:   reason,
		Response: response,
	})
	test.AssertError(t, err, "RevokeCertificate should've failed when certificate already revoked")
}

func TestRevokeCertificateNoResponse(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	err := features.Set(map[string]bool{features.ROCSPStage6.String(): true})
	test.AssertNotError(t, err, "failed to set features")
	defer features.Reset()

	reg := createWorkingRegistration(t, sa)
	// Add a cert to the DB to test with.
	certDER, err := os.ReadFile("www.eff.org.der")
	test.AssertNotError(t, err, "Couldn't read example cert DER")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   sa.clk.Now().UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add www.eff.org.der")

	serial := "000000000000000000000000000000021bd4"

	status, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "GetCertificateStatus failed")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusGood)

	fc.Add(1 * time.Hour)

	now := fc.Now()
	reason := int64(1)

	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial: serial,
		Date:   now.UnixNano(),
		Reason: reason,
	})
	test.AssertNotError(t, err, "RevokeCertificate should succeed with no response when ROCSPStage6 is enabled")
}

func TestUpdateRevokedCertificate(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// Add a cert to the DB to test with.
	reg := createWorkingRegistration(t, sa)
	certDER, err := os.ReadFile("www.eff.org.der")
	serial := "000000000000000000000000000000021bd4"
	issuedTime := fc.Now().UnixNano()
	test.AssertNotError(t, err, "Couldn't read example cert DER")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   issuedTime,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Couldn't add www.eff.org.der")
	fc.Add(1 * time.Hour)

	// Try to update it before its been revoked
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     fc.Now().UnixNano(),
		Backdate: fc.Now().UnixNano(),
		Reason:   ocsp.KeyCompromise,
		Response: []byte{4, 5, 6},
	})
	test.AssertError(t, err, "UpdateRevokedCertificate should have failed")
	test.AssertContains(t, err.Error(), "no certificate with serial")

	// Now revoke it, so we can update it.
	revokedTime := fc.Now().UnixNano()
	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     revokedTime,
		Reason:   ocsp.CessationOfOperation,
		Response: []byte{1, 2, 3},
	})
	test.AssertNotError(t, err, "RevokeCertificate failed")

	// Double check that setup worked.
	status, err := sa.GetCertificateStatus(ctx, &sapb.Serial{Serial: serial})
	test.AssertNotError(t, err, "GetCertificateStatus failed")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusRevoked)
	test.AssertEquals(t, int(status.RevokedReason), ocsp.CessationOfOperation)
	fc.Add(1 * time.Hour)

	// Try to update its revocation info with no backdate
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     fc.Now().UnixNano(),
		Reason:   ocsp.KeyCompromise,
		Response: []byte{4, 5, 6},
	})
	test.AssertError(t, err, "UpdateRevokedCertificate should have failed")
	test.AssertContains(t, err.Error(), "incomplete")

	// Try to update its revocation info for a reason other than keyCompromise
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     fc.Now().UnixNano(),
		Backdate: revokedTime,
		Reason:   ocsp.Unspecified,
		Response: []byte{4, 5, 6},
	})
	test.AssertError(t, err, "UpdateRevokedCertificate should have failed")
	test.AssertContains(t, err.Error(), "cannot update revocation for any reason other than keyCompromise")

	// Try to update the revocation info of the wrong certificate
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   "000000000000000000000000000000021bd5",
		Date:     fc.Now().UnixNano(),
		Backdate: revokedTime,
		Reason:   ocsp.KeyCompromise,
		Response: []byte{4, 5, 6},
	})
	test.AssertError(t, err, "UpdateRevokedCertificate should have failed")
	test.AssertContains(t, err.Error(), "no certificate with serial")

	// Try to update its revocation info with the wrong backdate
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     fc.Now().UnixNano(),
		Backdate: fc.Now().UnixNano(),
		Reason:   ocsp.KeyCompromise,
		Response: []byte{4, 5, 6},
	})
	test.AssertError(t, err, "UpdateRevokedCertificate should have failed")
	test.AssertContains(t, err.Error(), "no certificate with serial")

	// Try to update its revocation info correctly
	_, err = sa.UpdateRevokedCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   serial,
		Date:     fc.Now().UnixNano(),
		Backdate: revokedTime,
		Reason:   ocsp.KeyCompromise,
		Response: []byte{4, 5, 6},
	})
	test.AssertNotError(t, err, "UpdateRevokedCertificate failed")
}

func TestAddCertificateRenewalBit(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	reg := createWorkingRegistration(t, sa)

	// An example cert taken from EFF's website
	certDER, err := os.ReadFile("www.eff.org.der")
	test.AssertNotError(t, err, "Unexpected error reading www.eff.org.der test file")
	cert, err := x509.ParseCertificate(certDER)
	test.AssertNotError(t, err, "Unexpected error parsing www.eff.org.der test file")
	names := cert.DNSNames

	expires := fc.Now().Add(time.Hour * 2).UTC()
	issued := fc.Now()
	serial := "thrilla"

	// Add a FQDN set for the names so that it will be considered a renewal
	tx, err := sa.dbMap.Begin()
	test.AssertNotError(t, err, "Failed to open transaction")
	err = addFQDNSet(tx, names, serial, issued, expires)
	test.AssertNotError(t, err, "Failed to add name set")
	test.AssertNotError(t, tx.Commit(), "Failed to commit transaction")

	// Add the certificate with the same names.
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		Issued:   issued.UnixNano(),
		RegID:    reg.Id,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Failed to add precertificate")
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Failed to add certificate")

	assertIsRenewal := func(t *testing.T, name string, expected bool) {
		t.Helper()
		var count int
		err := sa.dbMap.SelectOne(
			&count,
			`SELECT COUNT(*) FROM issuedNames
		WHERE reversedName = ?
		AND renewal = ?`,
			ReverseName(name),
			expected,
		)
		test.AssertNotError(t, err, "Unexpected error from SelectOne on issuedNames")
		test.AssertEquals(t, count, 1)
	}

	// All of the names should have a issuedNames row marking it as a renewal.
	for _, name := range names {
		assertIsRenewal(t, name, true)
	}

	// Add a certificate with different names.
	certDER, err = os.ReadFile("test-cert.der")
	test.AssertNotError(t, err, "Unexpected error reading test-cert.der test file")
	cert, err = x509.ParseCertificate(certDER)
	test.AssertNotError(t, err, "Unexpected error parsing test-cert.der test file")
	names = cert.DNSNames

	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      certDER,
		Issued:   issued.UnixNano(),
		RegID:    reg.Id,
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "Failed to add precertificate")
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Failed to add certificate")

	// None of the names should have a issuedNames row marking it as a renewal.
	for _, name := range names {
		assertIsRenewal(t, name, false)
	}
}

func TestCountCertificatesRenewalBit(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// Create a test registration
	reg := createWorkingRegistration(t, sa)

	// Create a small throw away key for the test certificates.
	testKey, err := rsa.GenerateKey(rand.Reader, 512)
	test.AssertNotError(t, err, "error generating test key")

	// Create an initial test certificate for a set of domain names, issued an
	// hour ago.
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1337),
		DNSNames:              []string{"www.not-example.com", "not-example.com", "admin.not-example.com"},
		NotBefore:             fc.Now().Add(-time.Hour),
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	certADER, err := x509.CreateCertificate(rand.Reader, template, template, testKey.Public(), testKey)
	test.AssertNotError(t, err, "Failed to create test cert A")
	certA, _ := x509.ParseCertificate(certADER)

	// Update the template with a new serial number and a not before of now and
	// create a second test cert for the same names. This will be a renewal.
	template.SerialNumber = big.NewInt(7331)
	template.NotBefore = fc.Now()
	certBDER, err := x509.CreateCertificate(rand.Reader, template, template, testKey.Public(), testKey)
	test.AssertNotError(t, err, "Failed to create test cert B")
	certB, _ := x509.ParseCertificate(certBDER)

	// Update the template with a third serial number and a partially overlapping
	// set of names. This will not be a renewal but will help test the exact name
	// counts.
	template.SerialNumber = big.NewInt(0xC0FFEE)
	template.DNSNames = []string{"www.not-example.com"}
	certCDER, err := x509.CreateCertificate(rand.Reader, template, template, testKey.Public(), testKey)
	test.AssertNotError(t, err, "Failed to create test cert C")

	countName := func(t *testing.T, expectedName string) int64 {
		req := &sapb.CountCertificatesByNamesRequest{
			Names: []string{expectedName},
			Range: &sapb.Range{
				Earliest: fc.Now().Add(-5 * time.Hour).UnixNano(),
				Latest:   fc.Now().Add(5 * time.Hour).UnixNano(),
			},
		}
		counts, err := sa.CountCertificatesByNames(context.Background(), req)
		test.AssertNotError(t, err, "Unexpected err from CountCertificatesByNames")
		for name, count := range counts.Counts {
			if name == expectedName {
				return count
			}
		}
		return 0
	}

	// Add the first certificate - it won't be considered a renewal.
	issued := certA.NotBefore
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certADER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Failed to add CertA test certificate")

	// The count for the base domain should be 1 - just certA has been added.
	test.AssertEquals(t, countName(t, "not-example.com"), int64(1))

	// Add the second certificate - it should be considered a renewal
	issued = certB.NotBefore
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certBDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Failed to add CertB test certificate")

	// The count for the base domain should still be 1, just certA. CertB should
	// be ignored.
	test.AssertEquals(t, countName(t, "not-example.com"), int64(1))

	// Add the third certificate - it should not be considered a renewal
	_, err = sa.AddCertificate(ctx, &sapb.AddCertificateRequest{
		Der:    certCDER,
		RegID:  reg.Id,
		Issued: issued.UnixNano(),
	})
	test.AssertNotError(t, err, "Failed to add CertC test certificate")

	// The count for the base domain should be 2 now: certA and certC.
	// CertB should be ignored.
	test.AssertEquals(t, countName(t, "not-example.com"), int64(2))
}

func TestFinalizeAuthorization2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	fc.Set(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))

	authzID := createPendingAuthorization(t, sa, "aaa", fc.Now().Add(time.Hour))
	expires := fc.Now().Add(time.Hour * 2).UTC().UnixNano()
	attemptedAt := fc.Now().UnixNano()
	ip, _ := net.ParseIP("1.1.1.1").MarshalText()

	_, err := sa.FinalizeAuthorization2(context.Background(), &sapb.FinalizeAuthorizationRequest{
		Id: authzID,
		ValidationRecords: []*corepb.ValidationRecord{
			{
				Hostname:    "aaa",
				Port:        "123",
				Url:         "http://asd",
				AddressUsed: ip,
			},
		},
		Status:      string(core.StatusValid),
		Expires:     expires,
		Attempted:   string(core.ChallengeTypeHTTP01),
		AttemptedAt: attemptedAt,
	})
	test.AssertNotError(t, err, "sa.FinalizeAuthorization2 failed")

	dbVer, err := sa.GetAuthorization2(context.Background(), &sapb.AuthorizationID2{Id: authzID})
	test.AssertNotError(t, err, "sa.GetAuthorization2 failed")
	test.AssertEquals(t, dbVer.Status, string(core.StatusValid))
	test.AssertEquals(t, time.Unix(0, dbVer.Expires).UTC(), fc.Now().Add(time.Hour*2).UTC())
	test.AssertEquals(t, dbVer.Challenges[0].Status, string(core.StatusValid))
	test.AssertEquals(t, len(dbVer.Challenges[0].Validationrecords), 1)
	test.AssertEquals(t, time.Unix(0, dbVer.Challenges[0].Validated).UTC(), fc.Now().UTC())

	authzID = createPendingAuthorization(t, sa, "aaa", fc.Now().Add(time.Hour))
	prob, _ := bgrpc.ProblemDetailsToPB(probs.ConnectionFailure("it went bad captain"))

	_, err = sa.FinalizeAuthorization2(context.Background(), &sapb.FinalizeAuthorizationRequest{
		Id: authzID,
		ValidationRecords: []*corepb.ValidationRecord{
			{
				Hostname:    "aaa",
				Port:        "123",
				Url:         "http://asd",
				AddressUsed: ip,
			},
		},
		ValidationError: prob,
		Status:          string(core.StatusInvalid),
		Attempted:       string(core.ChallengeTypeHTTP01),
		Expires:         expires,
	})
	test.AssertNotError(t, err, "sa.FinalizeAuthorization2 failed")

	dbVer, err = sa.GetAuthorization2(context.Background(), &sapb.AuthorizationID2{Id: authzID})
	test.AssertNotError(t, err, "sa.GetAuthorization2 failed")
	test.AssertEquals(t, dbVer.Status, string(core.StatusInvalid))
	test.AssertEquals(t, dbVer.Challenges[0].Status, string(core.StatusInvalid))
	test.AssertEquals(t, len(dbVer.Challenges[0].Validationrecords), 1)
	test.AssertDeepEquals(t, dbVer.Challenges[0].Error, prob)
}

func TestGetPendingAuthorization2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	domain := "example.com"
	expiresA := fc.Now().Add(time.Hour).UTC()
	expiresB := fc.Now().Add(time.Hour * 3).UTC()
	authzIDA := createPendingAuthorization(t, sa, domain, expiresA)
	authzIDB := createPendingAuthorization(t, sa, domain, expiresB)

	regID := int64(1)
	validUntil := fc.Now().Add(time.Hour * 2).UTC().UnixNano()
	dbVer, err := sa.GetPendingAuthorization2(context.Background(), &sapb.GetPendingAuthorizationRequest{
		RegistrationID:  regID,
		IdentifierValue: domain,
		ValidUntil:      validUntil,
	})
	test.AssertNotError(t, err, "sa.GetPendingAuthorization2 failed")
	test.AssertEquals(t, fmt.Sprintf("%d", authzIDB), dbVer.Id)

	validUntil = fc.Now().UTC().UnixNano()
	dbVer, err = sa.GetPendingAuthorization2(context.Background(), &sapb.GetPendingAuthorizationRequest{
		RegistrationID:  regID,
		IdentifierValue: domain,
		ValidUntil:      validUntil,
	})
	test.AssertNotError(t, err, "sa.GetPendingAuthorization2 failed")
	test.AssertEquals(t, fmt.Sprintf("%d", authzIDA), dbVer.Id)
}

func TestCountPendingAuthorizations2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	expiresA := fc.Now().Add(time.Hour).UTC()
	expiresB := fc.Now().Add(time.Hour * 3).UTC()
	_ = createPendingAuthorization(t, sa, "example.com", expiresA)
	_ = createPendingAuthorization(t, sa, "example.com", expiresB)

	// Registration has two new style pending authorizations
	regID := int64(1)
	count, err := sa.CountPendingAuthorizations2(context.Background(), &sapb.RegistrationID{
		Id: regID,
	})
	test.AssertNotError(t, err, "sa.CountPendingAuthorizations2 failed")
	test.AssertEquals(t, count.Count, int64(2))

	// Registration has two new style pending authorizations, one of which has expired
	fc.Add(time.Hour * 2)
	count, err = sa.CountPendingAuthorizations2(context.Background(), &sapb.RegistrationID{
		Id: regID,
	})
	test.AssertNotError(t, err, "sa.CountPendingAuthorizations2 failed")
	test.AssertEquals(t, count.Count, int64(1))

	// Registration with no authorizations should be 0
	noReg := int64(20)
	count, err = sa.CountPendingAuthorizations2(context.Background(), &sapb.RegistrationID{
		Id: noReg,
	})
	test.AssertNotError(t, err, "sa.CountPendingAuthorizations2 failed")
	test.AssertEquals(t, count.Count, int64(0))
}

func TestAuthzModelMapToPB(t *testing.T) {
	baseExpires := time.Now()
	input := map[string]authzModel{
		"example.com": {
			ID:              123,
			IdentifierType:  0,
			IdentifierValue: "example.com",
			RegistrationID:  77,
			Status:          1,
			Expires:         baseExpires,
			Challenges:      4,
		},
		"www.example.com": {
			ID:              124,
			IdentifierType:  0,
			IdentifierValue: "www.example.com",
			RegistrationID:  77,
			Status:          1,
			Expires:         baseExpires,
			Challenges:      1,
		},
		"other.example.net": {
			ID:              125,
			IdentifierType:  0,
			IdentifierValue: "other.example.net",
			RegistrationID:  77,
			Status:          1,
			Expires:         baseExpires,
			Challenges:      3,
		},
	}

	out, err := authzModelMapToPB(input)
	if err != nil {
		t.Fatal(err)
	}

	for _, el := range out.Authz {
		model, ok := input[el.Domain]
		if !ok {
			t.Errorf("output had element for %q, a hostname not present in input", el.Domain)
		}
		authzPB := el.Authz
		test.AssertEquals(t, authzPB.Id, fmt.Sprintf("%d", model.ID))
		test.AssertEquals(t, authzPB.Identifier, model.IdentifierValue)
		test.AssertEquals(t, authzPB.RegistrationID, model.RegistrationID)
		test.AssertEquals(t, authzPB.Status, string(uintToStatus[model.Status]))
		gotTime := time.Unix(0, authzPB.Expires).UTC()
		if !model.Expires.Equal(gotTime) {
			t.Errorf("Times didn't match. Got %s, expected %s (%d)", gotTime, model.Expires, authzPB.Expires)
		}
		if len(el.Authz.Challenges) != bits.OnesCount(uint(model.Challenges)) {
			t.Errorf("wrong number of challenges for %q: got %d, expected %d", el.Domain,
				len(el.Authz.Challenges), bits.OnesCount(uint(model.Challenges)))
		}
		switch model.Challenges {
		case 1:
			test.AssertEquals(t, el.Authz.Challenges[0].Type, "http-01")
		case 3:
			test.AssertEquals(t, el.Authz.Challenges[0].Type, "http-01")
			test.AssertEquals(t, el.Authz.Challenges[1].Type, "dns-01")
		case 4:
			test.AssertEquals(t, el.Authz.Challenges[0].Type, "tls-alpn-01")
		}

		delete(input, el.Domain)
	}

	for k := range input {
		t.Errorf("hostname %q was not present in output", k)
	}
}

func TestGetValidOrderAuthorizations2(t *testing.T) {
	sa, fc, cleanup := initSA(t)
	defer cleanup()

	// Create two new valid authorizations
	reg := createWorkingRegistration(t, sa)
	identA := "a.example.com"
	identB := "b.example.com"
	expires := fc.Now().Add(time.Hour * 24 * 7).UTC()
	attemptedAt := fc.Now()

	authzIDA := createFinalizedAuthorization(t, sa, identA, expires, "valid", attemptedAt)
	authzIDB := createFinalizedAuthorization(t, sa, identB, expires, "valid", attemptedAt)

	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          fc.Now().Truncate(time.Second).UnixNano(),
			Names:            []string{"a.example.com", "b.example.com"},
			V2Authorizations: []int64{authzIDA, authzIDB},
		},
	})
	test.AssertNotError(t, err, "AddOrder failed")

	authzMap, err := sa.GetValidOrderAuthorizations2(
		context.Background(),
		&sapb.GetValidOrderAuthorizationsRequest{
			Id:     order.Id,
			AcctID: reg.Id,
		})
	test.AssertNotError(t, err, "sa.GetValidOrderAuthorizations failed")
	test.AssertNotNil(t, authzMap, "sa.GetValidOrderAuthorizations result was nil")
	test.AssertEquals(t, len(authzMap.Authz), 2)

	namesToCheck := map[string]int64{"a.example.com": authzIDA, "b.example.com": authzIDB}
	for _, a := range authzMap.Authz {
		if fmt.Sprintf("%d", namesToCheck[a.Authz.Identifier]) != a.Authz.Id {
			t.Fatalf("incorrect identifier %q with id %s", a.Authz.Identifier, a.Authz.Id)
		}
		test.AssertEquals(t, a.Authz.Expires, expires.UnixNano())
		delete(namesToCheck, a.Authz.Identifier)
	}

	// Getting the order authorizations for an order that doesn't exist should return nothing
	missingID := int64(0xC0FFEEEEEEE)
	authzMap, err = sa.GetValidOrderAuthorizations2(
		context.Background(),
		&sapb.GetValidOrderAuthorizationsRequest{
			Id:     missingID,
			AcctID: reg.Id,
		})
	test.AssertNotError(t, err, "sa.GetValidOrderAuthorizations failed")
	test.AssertEquals(t, len(authzMap.Authz), 0)

	// Getting the order authorizations for an order that does exist, but for the
	// wrong acct ID should return nothing
	wrongAcctID := int64(0xDEADDA7ABA5E)
	authzMap, err = sa.GetValidOrderAuthorizations2(
		context.Background(),
		&sapb.GetValidOrderAuthorizationsRequest{
			Id:     order.Id,
			AcctID: wrongAcctID,
		})
	test.AssertNotError(t, err, "sa.GetValidOrderAuthorizations failed")
	test.AssertEquals(t, len(authzMap.Authz), 0)
}

func TestCountInvalidAuthorizations2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// Create two authorizations, one pending, one invalid
	fc.Add(time.Hour)
	reg := createWorkingRegistration(t, sa)
	ident := "aaa"
	expiresA := fc.Now().Add(time.Hour).UTC()
	expiresB := fc.Now().Add(time.Hour * 3).UTC()
	attemptedAt := fc.Now()
	_ = createFinalizedAuthorization(t, sa, ident, expiresA, "invalid", attemptedAt)
	_ = createPendingAuthorization(t, sa, ident, expiresB)

	earliest, latest := fc.Now().Add(-time.Hour).UTC().UnixNano(), fc.Now().Add(time.Hour*5).UTC().UnixNano()
	count, err := sa.CountInvalidAuthorizations2(context.Background(), &sapb.CountInvalidAuthorizationsRequest{
		RegistrationID: reg.Id,
		Hostname:       ident,
		Range: &sapb.Range{
			Earliest: earliest,
			Latest:   latest,
		},
	})
	test.AssertNotError(t, err, "sa.CountInvalidAuthorizations2 failed")
	test.AssertEquals(t, count.Count, int64(1))
}

func TestGetValidAuthorizations2(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	// Create a valid authorization
	ident := "aaa"
	expires := fc.Now().Add(time.Hour).UTC()
	attemptedAt := fc.Now()
	authzID := createFinalizedAuthorization(t, sa, ident, expires, "valid", attemptedAt)

	now := fc.Now().UTC().UnixNano()
	regID := int64(1)
	authzs, err := sa.GetValidAuthorizations2(context.Background(), &sapb.GetValidAuthorizationsRequest{
		Domains: []string{
			"aaa",
			"bbb",
		},
		RegistrationID: regID,
		Now:            now,
	})
	test.AssertNotError(t, err, "sa.GetValidAuthorizations2 failed")
	test.AssertEquals(t, len(authzs.Authz), 1)
	test.AssertEquals(t, authzs.Authz[0].Domain, ident)
	test.AssertEquals(t, authzs.Authz[0].Authz.Id, fmt.Sprintf("%d", authzID))
}

func TestGetOrderExpired(t *testing.T) {
	sa, fc, cleanUp := initSA(t)
	defer cleanUp()

	fc.Add(time.Hour * 5)
	reg := createWorkingRegistration(t, sa)
	order, err := sa.NewOrderAndAuthzs(context.Background(), &sapb.NewOrderAndAuthzsRequest{
		NewOrder: &sapb.NewOrderRequest{
			RegistrationID:   reg.Id,
			Expires:          fc.Now().Add(-time.Hour).UnixNano(),
			Names:            []string{"example.com"},
			V2Authorizations: []int64{666},
		},
	})
	test.AssertNotError(t, err, "NewOrderAndAuthzs failed")
	_, err = sa.GetOrder(context.Background(), &sapb.OrderRequest{
		Id: order.Id,
	})
	test.AssertError(t, err, "GetOrder didn't fail for an expired order")
	test.AssertErrorIs(t, err, berrors.NotFound)
}

func TestBlockedKey(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	hashA := make([]byte, 32)
	hashA[0] = 1
	hashB := make([]byte, 32)
	hashB[0] = 2

	added := time.Now().UnixNano()
	source := "API"
	_, err := sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash: hashA,
		Added:   added,
		Source:  source,
	})
	test.AssertNotError(t, err, "AddBlockedKey failed")
	_, err = sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash: hashA,
		Added:   added,
		Source:  source,
	})
	test.AssertNotError(t, err, "AddBlockedKey failed with duplicate insert")

	comment := "testing comments"
	_, err = sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash: hashB,
		Added:   added,
		Source:  source,
		Comment: comment,
	})
	test.AssertNotError(t, err, "AddBlockedKey failed")

	exists, err := sa.KeyBlocked(context.Background(), &sapb.KeyBlockedRequest{
		KeyHash: hashA,
	})
	test.AssertNotError(t, err, "KeyBlocked failed")
	test.Assert(t, exists != nil, "*sapb.Exists is nil")
	test.Assert(t, exists.Exists, "KeyBlocked returned false for blocked key")
	exists, err = sa.KeyBlocked(context.Background(), &sapb.KeyBlockedRequest{
		KeyHash: hashB,
	})
	test.AssertNotError(t, err, "KeyBlocked failed")
	test.Assert(t, exists != nil, "*sapb.Exists is nil")
	test.Assert(t, exists.Exists, "KeyBlocked returned false for blocked key")
	exists, err = sa.KeyBlocked(context.Background(), &sapb.KeyBlockedRequest{
		KeyHash: []byte{5},
	})
	test.AssertNotError(t, err, "KeyBlocked failed")
	test.Assert(t, exists != nil, "*sapb.Exists is nil")
	test.Assert(t, !exists.Exists, "KeyBlocked returned true for non-blocked key")
}

func TestAddBlockedKeyUnknownSource(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	_, err := sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash: []byte{1, 2, 3},
		Added:   1,
		Source:  "heyo",
	})
	test.AssertError(t, err, "AddBlockedKey didn't fail with unknown source")
	test.AssertEquals(t, err.Error(), "unknown source")
}

func TestBlockedKeyRevokedBy(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	_, err := sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash: []byte{1},
		Added:   1,
		Source:  "API",
	})
	test.AssertNotError(t, err, "AddBlockedKey failed")

	_, err = sa.AddBlockedKey(context.Background(), &sapb.AddBlockedKeyRequest{
		KeyHash:   []byte{2},
		Added:     1,
		Source:    "API",
		RevokedBy: 1,
	})
	test.AssertNotError(t, err, "AddBlockedKey failed")
}

func TestHashNames(t *testing.T) {
	// Test that it is deterministic
	h1 := HashNames([]string{"a"})
	h2 := HashNames([]string{"a"})
	test.AssertByteEquals(t, h1, h2)

	// Test that it differentiates
	h1 = HashNames([]string{"a"})
	h2 = HashNames([]string{"b"})
	test.Assert(t, !bytes.Equal(h1, h2), "Should have been different")

	// Test that it is not subject to ordering
	h1 = HashNames([]string{"a", "b"})
	h2 = HashNames([]string{"b", "a"})
	test.AssertByteEquals(t, h1, h2)

	// Test that it is not subject to case
	h1 = HashNames([]string{"a", "b"})
	h2 = HashNames([]string{"A", "B"})
	test.AssertByteEquals(t, h1, h2)

	// Test that it is not subject to duplication
	h1 = HashNames([]string{"a", "a"})
	h2 = HashNames([]string{"a"})
	test.AssertByteEquals(t, h1, h2)
}

func TestIncidentsForSerial(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	testSADbMap, err := NewDbMap(vars.DBConnSAFullPerms, DbSettings{})
	test.AssertNotError(t, err, "Couldn't create test dbMap")

	testIncidentsDbMap, err := NewDbMap(vars.DBConnIncidentsFullPerms, DbSettings{})
	test.AssertNotError(t, err, "Couldn't create test dbMap")
	defer test.ResetIncidentsTestDatabase(t)

	// Add a disabled incident.
	err = testSADbMap.Insert(&incidentModel{
		SerialTable: "incident_foo",
		URL:         "https://example.com/foo-incident",
		RenewBy:     sa.clk.Now().Add(time.Hour * 24 * 7),
		Enabled:     false,
	})
	test.AssertNotError(t, err, "Failed to insert disabled incident")

	// No incidents are enabled, so this should return in error.
	result, err := sa.IncidentsForSerial(context.Background(), &sapb.Serial{Serial: "1337"})
	test.AssertNotError(t, err, "fetching from no incidents")
	test.AssertEquals(t, len(result.Incidents), 0)

	// Add an enabled incident.
	err = testSADbMap.Insert(&incidentModel{
		SerialTable: "incident_bar",
		URL:         "https://example.com/test-incident",
		RenewBy:     sa.clk.Now().Add(time.Hour * 24 * 7),
		Enabled:     true,
	})
	test.AssertNotError(t, err, "Failed to insert enabled incident")

	// Add a row to the incident table with serial '1338'.
	affectedCertA := incidentSerialModel{
		Serial:         "1338",
		RegistrationID: 1,
		OrderID:        1,
		LastNoticeSent: sa.clk.Now().Add(time.Hour * 24 * 7),
	}
	_, err = testIncidentsDbMap.Exec(
		fmt.Sprintf("INSERT INTO incident_bar (%s) VALUES ('%s', %d, %d, '%s')",
			"serial, registrationID, orderID, lastNoticeSent",
			affectedCertA.Serial,
			affectedCertA.RegistrationID,
			affectedCertA.OrderID,
			affectedCertA.LastNoticeSent.Format("2006-01-02 15:04:05"),
		),
	)
	test.AssertNotError(t, err, "Error while inserting row for '1338' into incident table")

	// The incident table should not contain a row with serial '1337'.
	result, err = sa.IncidentsForSerial(context.Background(), &sapb.Serial{Serial: "1337"})
	test.AssertNotError(t, err, "fetching from one incident")
	test.AssertEquals(t, len(result.Incidents), 0)

	// Add a row to the incident table with serial '1337'.
	affectedCertB := incidentSerialModel{
		Serial:         "1337",
		RegistrationID: 2,
		OrderID:        2,
		LastNoticeSent: sa.clk.Now().Add(time.Hour * 24 * 7),
	}
	_, err = testIncidentsDbMap.Exec(
		fmt.Sprintf("INSERT INTO incident_bar (%s) VALUES ('%s', %d, %d, '%s')",
			"serial, registrationID, orderID, lastNoticeSent",
			affectedCertB.Serial,
			affectedCertB.RegistrationID,
			affectedCertB.OrderID,
			affectedCertB.LastNoticeSent.Format("2006-01-02 15:04:05"),
		),
	)
	test.AssertNotError(t, err, "Error while inserting row for '1337' into incident table")

	// The incident table should now contain a row with serial '1337'.
	result, err = sa.IncidentsForSerial(context.Background(), &sapb.Serial{Serial: "1337"})
	test.AssertNotError(t, err, "Failed to retrieve incidents for serial")
	test.AssertEquals(t, len(result.Incidents), 1)
}

type mockSerialsForIncidentServerStream struct {
	grpc.ServerStream
	output chan<- *sapb.IncidentSerial
}

func (s mockSerialsForIncidentServerStream) Send(serial *sapb.IncidentSerial) error {
	s.output <- serial
	return nil
}

func (s mockSerialsForIncidentServerStream) Context() context.Context {
	return context.Background()
}

func TestSerialsForIncident(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	testIncidentsDbMap, err := NewDbMap(vars.DBConnIncidentsFullPerms, DbSettings{})
	test.AssertNotError(t, err, "Couldn't create test dbMap")
	defer test.ResetIncidentsTestDatabase(t)

	// Request serials from a malformed incident table name.
	mockServerStream := mockSerialsForIncidentServerStream{}
	err = sa.SerialsForIncident(
		&sapb.SerialsForIncidentRequest{
			IncidentTable: "incidesnt_Baz",
		},
		mockServerStream,
	)
	test.AssertError(t, err, "Expected error for malformed table name")
	test.AssertContains(t, err.Error(), "malformed table name \"incidesnt_Baz\"")

	// Request serials from another malformed incident table name.
	mockServerStream = mockSerialsForIncidentServerStream{}
	longTableName := "incident_l" + strings.Repeat("o", 1000) + "ng"
	err = sa.SerialsForIncident(
		&sapb.SerialsForIncidentRequest{
			IncidentTable: longTableName,
		},
		mockServerStream,
	)
	test.AssertError(t, err, "Expected error for long table name")
	test.AssertContains(t, err.Error(), fmt.Sprintf("malformed table name %q", longTableName))

	// Request serials for an incident table which doesn't exists.
	mockServerStream = mockSerialsForIncidentServerStream{}
	err = sa.SerialsForIncident(
		&sapb.SerialsForIncidentRequest{
			IncidentTable: "incident_baz",
		},
		mockServerStream,
	)
	test.AssertError(t, err, "Expected error for nonexistent table name")

	// Assert that the error is a MySQL error so we can inspect the error code.
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		// We expect the error code to be 1146 (ER_NO_SUCH_TABLE):
		// https://mariadb.com/kb/en/mariadb-error-codes/
		test.AssertEquals(t, mysqlErr.Number, uint16(1146))
	} else {
		t.Fatalf("Expected MySQL Error 1146 (ER_NO_SUCH_TABLE) from Recv(), got %q", err)
	}

	// Request serials from table 'incident_foo', which we expect to exist but
	// be empty.
	stream := make(chan *sapb.IncidentSerial)
	mockServerStream = mockSerialsForIncidentServerStream{output: stream}
	go func() {
		err = sa.SerialsForIncident(
			&sapb.SerialsForIncidentRequest{
				IncidentTable: "incident_foo",
			},
			mockServerStream,
		)
		close(stream) // Let our main test thread continue.
	}()
	for range stream {
		t.Fatal("No serials should have been written to this stream")
	}
	test.AssertNotError(t, err, "Error calling SerialsForIncident on empty table")

	// Add 4 rows of incident serials to 'incident_foo'.
	expectedSerials := map[string]bool{
		"1335": true, "1336": true, "1337": true, "1338": true,
	}
	for i := range expectedSerials {
		mrand.Seed(time.Now().Unix())
		randInt := func() int64 { return mrand.Int63() }
		_, err := testIncidentsDbMap.Exec(
			fmt.Sprintf("INSERT INTO incident_foo (%s) VALUES ('%s', %d, %d, '%s')",
				"serial, registrationID, orderID, lastNoticeSent",
				i,
				randInt(),
				randInt(),
				sa.clk.Now().Add(time.Hour*24*7).Format("2006-01-02 15:04:05"),
			),
		)
		test.AssertNotError(t, err, fmt.Sprintf("Error while inserting row for '%s' into incident table", i))
	}

	// Request all 4 serials from the incident table we just added entries to.
	stream = make(chan *sapb.IncidentSerial)
	mockServerStream = mockSerialsForIncidentServerStream{output: stream}
	go func() {
		err = sa.SerialsForIncident(
			&sapb.SerialsForIncidentRequest{
				IncidentTable: "incident_foo",
			},
			mockServerStream,
		)
		close(stream)
	}()
	receivedSerials := make(map[string]bool)
	for serial := range stream {
		if len(receivedSerials) > 4 {
			t.Fatal("Received too many serials")
		}
		if _, ok := receivedSerials[serial.Serial]; ok {
			t.Fatalf("Received serial %q more than once", serial.Serial)
		}
		receivedSerials[serial.Serial] = true
	}
	test.AssertDeepEquals(t, receivedSerials, map[string]bool{
		"1335": true, "1336": true, "1337": true, "1338": true,
	})
	test.AssertNotError(t, err, "Error getting serials for incident")
}

type mockGetRevokedCertsServerStream struct {
	grpc.ServerStream
	output chan<- *corepb.CRLEntry
}

func (s mockGetRevokedCertsServerStream) Send(entry *corepb.CRLEntry) error {
	s.output <- entry
	return nil
}

func (s mockGetRevokedCertsServerStream) Context() context.Context {
	return context.Background()
}

func TestGetRevokedCerts(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	// Add a cert to the DB to test with. We use AddPrecertificate because it sets
	// up the certificateStatus row we need. This particular cert has a notAfter
	// date of Mar 6 2023, and we lie about its IssuerNameID to make things easy.
	reg := createWorkingRegistration(t, sa)
	eeCert, err := core.LoadCert("../test/hierarchy/ee-e1.cert.pem")
	test.AssertNotError(t, err, "failed to load test cert")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      eeCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   eeCert.NotBefore.UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "failed to add test cert")

	// Check that it worked.
	status, err := sa.GetCertificateStatus(
		ctx, &sapb.Serial{Serial: core.SerialToString(eeCert.SerialNumber)})
	test.AssertNotError(t, err, "GetCertificateStatus failed")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusGood)

	// Here's a little helper func we'll use to call GetRevokedCerts and count
	// how many results it returned.
	countRevokedCerts := func(req *sapb.GetRevokedCertsRequest) (int, error) {
		stream := make(chan *corepb.CRLEntry)
		mockServerStream := mockGetRevokedCertsServerStream{output: stream}
		var err error
		go func() {
			err = sa.GetRevokedCerts(req, mockServerStream)
			close(stream)
		}()
		entriesReceived := 0
		for range stream {
			entriesReceived++
		}
		return entriesReceived, err
	}

	// Asking for revoked certs now should return no results.
	count, err := countRevokedCerts(&sapb.GetRevokedCertsRequest{
		IssuerNameID:  1,
		ExpiresAfter:  time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		ExpiresBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		RevokedBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	})
	test.AssertNotError(t, err, "zero rows shouldn't result in error")
	test.AssertEquals(t, count, 0)

	// Revoke the certificate.
	_, err = sa.RevokeCertificate(context.Background(), &sapb.RevokeCertificateRequest{
		Serial:   core.SerialToString(eeCert.SerialNumber),
		Date:     time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		Reason:   1,
		Response: []byte{1, 2, 3},
	})
	test.AssertNotError(t, err, "failed to revoke test cert")

	// Asking for revoked certs now should return one result.
	count, err = countRevokedCerts(&sapb.GetRevokedCertsRequest{
		IssuerNameID:  1,
		ExpiresAfter:  time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		ExpiresBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		RevokedBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	})
	test.AssertNotError(t, err, "normal usage shouldn't result in error")
	test.AssertEquals(t, count, 1)

	// Asking for revoked certs with an old RevokedBefore should return no results.
	count, err = countRevokedCerts(&sapb.GetRevokedCertsRequest{
		IssuerNameID:  1,
		ExpiresAfter:  time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		ExpiresBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		RevokedBefore: time.Date(2020, time.March, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	})
	test.AssertNotError(t, err, "zero rows shouldn't result in error")
	test.AssertEquals(t, count, 0)

	// Asking for revoked certs in a time period that does not cover this cert's
	// notAfter timestamp should return zero results.
	count, err = countRevokedCerts(&sapb.GetRevokedCertsRequest{
		IssuerNameID:  1,
		ExpiresAfter:  time.Date(2022, time.March, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		ExpiresBefore: time.Date(2022, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
		RevokedBefore: time.Date(2023, time.April, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	})
	test.AssertNotError(t, err, "zero rows shouldn't result in error")
	test.AssertEquals(t, count, 0)
}

func TestGetMaxExpiration(t *testing.T) {
	sa, _, cleanUp := initSA(t)
	defer cleanUp()

	// Add a cert to the DB to test with. We use AddPrecertificate because it sets
	// up the certificateStatus row we need. This particular cert has a notAfter
	// date of Mar 6 2023, and we lie about its IssuerNameID to make things easy.
	reg := createWorkingRegistration(t, sa)
	eeCert, err := core.LoadCert("../test/hierarchy/ee-e1.cert.pem")
	test.AssertNotError(t, err, "failed to load test cert")
	_, err = sa.AddPrecertificate(ctx, &sapb.AddCertificateRequest{
		Der:      eeCert.Raw,
		RegID:    reg.Id,
		Ocsp:     nil,
		Issued:   eeCert.NotBefore.UnixNano(),
		IssuerID: 1,
	})
	test.AssertNotError(t, err, "failed to add test cert")

	lastExpiry, err := sa.GetMaxExpiration(context.Background(), &emptypb.Empty{})
	test.AssertNotError(t, err, "getting last expriy should succeed")
	test.Assert(t, lastExpiry.AsTime().Equal(eeCert.NotAfter), "times should be equal")
}
