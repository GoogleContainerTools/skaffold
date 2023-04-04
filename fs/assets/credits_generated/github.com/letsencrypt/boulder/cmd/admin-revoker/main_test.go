package notmain

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/user"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	akamaipb "github.com/letsencrypt/boulder/akamai/proto"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/goodkey"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/mocks"
	"github.com/letsencrypt/boulder/ra"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/test"
	ira "github.com/letsencrypt/boulder/test/inmem/ra"
	isa "github.com/letsencrypt/boulder/test/inmem/sa"
	"github.com/letsencrypt/boulder/test/vars"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockOCSPA struct {
	mocks.MockCA
}

func (ca *mockOCSPA) GenerateOCSP(context.Context, *capb.GenerateOCSPRequest, ...grpc.CallOption) (*capb.OCSPResponse, error) {
	return &capb.OCSPResponse{Response: []byte("fakeocspbytes")}, nil
}

type mockPurger struct{}

func (mp *mockPurger) Purge(context.Context, *akamaipb.PurgeRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func TestRevokeSerialBatchFile(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	entries, _ := setupUniqueTestEntries(t)
	testCtx.createAndRegisterEntries(t, entries)

	serialFile, err := os.CreateTemp("", "serials")
	test.AssertNotError(t, err, "failed to open temp file")
	defer os.Remove(serialFile.Name())

	for _, e := range entries {
		_, err = serialFile.WriteString(fmt.Sprintf("%s\n", core.SerialToString(e.serial)))
		test.AssertNotError(t, err, "failed to write serial to temp file")
	}
	err = testCtx.revoker.revokeSerialBatchFile(context.Background(), serialFile.Name(), 0, 2)
	test.AssertNotError(t, err, "revokeBatch failed")

	for _, e := range entries {
		status, err := testCtx.ssa.GetCertificateStatus(context.Background(), &sapb.Serial{Serial: core.SerialToString(e.serial)})
		test.AssertNotError(t, err, "failed to retrieve certificate status")
		test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusRevoked)
	}
}

func TestRevokeIncidentTableSerials(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	entries, _ := setupUniqueTestEntries(t)
	testCtx.createAndRegisterEntries(t, entries)

	testIncidentsDbMap, err := sa.NewDbMap(vars.DBConnIncidentsFullPerms, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't create test dbMap")

	// Ensure that an empty incident table results in the expected log output.
	err = testCtx.revoker.revokeIncidentTableSerials(context.Background(), "incident_foo", 0, 1)
	test.AssertNotError(t, err, "revokeIncidentTableSerials failed")
	test.Assert(t, len(testCtx.log.GetAllMatching("No serials found in incident table")) > 0, "Expected log output not found")
	testCtx.log.Clear()

	_, err = testIncidentsDbMap.Exec(
		fmt.Sprintf("INSERT INTO incident_foo (%s) VALUES ('%s', %d, %d, '%s')",
			"serial, registrationID, orderID, lastNoticeSent",
			core.SerialToString(entries[0].serial),
			entries[0].regId,
			42,
			testCtx.revoker.clk.Now().Add(-time.Hour*24*7).Format("2006-01-02 15:04:05"),
		),
	)
	test.AssertNotError(t, err, "while inserting row into incident table")

	err = testCtx.revoker.revokeIncidentTableSerials(context.Background(), "incident_foo", 0, 1)
	test.AssertNotError(t, err, "revokeIncidentTableSerials failed")

	// Ensure that a populated incident table results in the expected log output.
	test.AssertNotError(t, err, "revokeIncidentTableSerials failed")
	test.Assert(t, len(testCtx.log.GetAllMatching("No serials found in incident table")) <= 0, "Expected log output not found")

	status, err := testCtx.ssa.GetCertificateStatus(context.Background(), &sapb.Serial{Serial: core.SerialToString(entries[0].serial)})
	test.AssertNotError(t, err, "failed to retrieve certificate status")
	test.AssertEquals(t, core.OCSPStatus(status.Status), core.OCSPStatusRevoked)
}

func TestBlockAndRevokeByPrivateKey(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	uniqueEntries, duplicateEntry := setupUniqueTestEntries(t)
	testCtx.createAndRegisterEntries(t, uniqueEntries)
	testCtx.createAndRegisterEntry(t, duplicateEntry)

	// Write the key contents of our duplicate entry to a temp file.
	duplicateKeyFile, err := os.CreateTemp("", "key")
	test.AssertNotError(t, err, "failed to create temp file")
	der, err := x509.MarshalPKCS8PrivateKey(duplicateEntry.testKey)
	test.AssertNotError(t, err, "failed to marshal testKey1 to DER")
	err = pem.Encode(duplicateKeyFile,
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		},
	)
	test.AssertNotError(t, err, "failed to PEM encode test key 1")
	test.AssertNotError(t, err, "failed to write to temp file")
	defer os.Remove(duplicateKeyFile.Name())

	// Get the SPKI hash for the provided keypair.
	spkiHash, err := getPublicKeySPKIHash(&duplicateEntry.testKey.PublicKey)
	test.AssertNotError(t, err, "Failed to get SPKI hash for dupe.")

	// Ensure that the SPKI hash hasn't already been added to the blockedKeys
	// table.
	keyExists, err := testCtx.revoker.spkiHashInBlockedKeys(spkiHash)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.Assert(t, !keyExists, "SPKI hash should not be in blockedKeys")

	// For some additional validation let's ensure that counts for all test
	// entries, except our known duplicate, are 1.
	for _, e := range uniqueEntries {
		switch e.names[0] {
		case uniqueEntries[0].names[0]:
			// example-1337.com
			count, err := testCtx.revoker.countCertsMatchingSPKIHash(e.spkiHash)
			test.AssertNotError(t, err, "countCertsMatchingSPKIHash for entry failed")
			test.AssertEquals(t, count, 2)

		case uniqueEntries[1].names[0]:
			// example-1338.com
			count, err := testCtx.revoker.countCertsMatchingSPKIHash(e.spkiHash)
			test.AssertNotError(t, err, "countCertsMatchingSPKIHash for entry failed")
			test.AssertEquals(t, count, 1)

		case uniqueEntries[2].names[0]:
			// example-1339.com
			count, err := testCtx.revoker.countCertsMatchingSPKIHash(e.spkiHash)
			test.AssertNotError(t, err, "countCertsMatchingSPKIHash for entry failed")
			test.AssertEquals(t, count, 1)
		}
	}

	// Revoke one of our two duplicate certificates by serial. This is to test
	// that revokeByPrivateKey will continue if one of the two matching
	// certificates has already been revoked.
	err = testCtx.revoker.revokeBySerial(context.Background(), core.SerialToString(duplicateEntry.serial), 1, true)
	test.AssertNotError(t, err, "While attempting to revoke 1 of our matching certificates ahead of time")

	// Revoke the certificates, but do not block issuance.
	err = testCtx.revoker.revokeByPrivateKey(context.Background(), duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to revoke certificates for the provided key")

	// Ensure that the key is not blocked, yet.
	keyExists, err = testCtx.revoker.spkiHashInBlockedKeys(spkiHash)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.Assert(t, !keyExists, "SPKI hash should not be in blockedKeys")

	// Block issuance for the key.
	err = testCtx.revoker.blockByPrivateKey(context.Background(), "", duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")

	// Ensure that the key is now blocked.
	keyExists, err = testCtx.revoker.spkiHashInBlockedKeys(spkiHash)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.Assert(t, keyExists, "SPKI hash should not be in blockedKeys")

	// Ensure that blocking issuance is idempotent.
	err = testCtx.revoker.blockByPrivateKey(context.Background(), "", duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")
}

func TestPrivateKeyBlock(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	uniqueEntries, duplicateEntry := setupUniqueTestEntries(t)
	testCtx.createAndRegisterEntries(t, uniqueEntries)
	testCtx.createAndRegisterEntry(t, duplicateEntry)

	// Write the key contents of our duplicate entry to a temp file.
	duplicateKeyFile, err := os.CreateTemp("", "key")
	test.AssertNotError(t, err, "failed to create temp file")
	der, err := x509.MarshalPKCS8PrivateKey(duplicateEntry.testKey)
	test.AssertNotError(t, err, "failed to marshal testKey1 to DER")
	err = pem.Encode(duplicateKeyFile,
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		},
	)
	test.AssertNotError(t, err, "failed to PEM encode test key")
	test.AssertNotError(t, err, "failed to write to temp file")
	defer os.Remove(duplicateKeyFile.Name())

	// Get the SPKI hash for the provided keypair.
	duplicateKeySPKI, err := getPublicKeySPKIHash(&duplicateEntry.testKey.PublicKey)
	test.AssertNotError(t, err, "Failed to get SPKI hash for dupe.")

	// Query the 'keyHashToSerial' table for certificates with a matching SPKI
	// hash. We expect that since this key was re-used we'll find 2 matches.
	count, err := testCtx.revoker.countCertsMatchingSPKIHash(duplicateKeySPKI)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.AssertEquals(t, count, 2)

	// With dryRun=true this should not block the key.
	err = privateKeyBlock(&testCtx.revoker, true, "", count, duplicateKeySPKI, duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")

	// Ensure that the key is not blocked, yet.
	keyExists, err := testCtx.revoker.spkiHashInBlockedKeys(duplicateKeySPKI)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.Assert(t, !keyExists, "SPKI hash should not be in blockedKeys")

	// With dryRun=false this should block the key.
	comment := "key blocked as part of test"
	err = privateKeyBlock(&testCtx.revoker, false, comment, count, duplicateKeySPKI, duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")

	// With dryRun=false this should result in an error as the key is already blocked.
	err = privateKeyBlock(&testCtx.revoker, false, "", count, duplicateKeySPKI, duplicateKeyFile.Name())
	test.AssertError(t, err, "Attempting to block a key which is already blocked should have failed.")

	// Ensure that the key is now blocked.
	keyExists, err = testCtx.revoker.spkiHashInBlockedKeys(duplicateKeySPKI)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.Assert(t, keyExists, "SPKI hash should not be in blockedKeys")

	// Ensure that the comment was set as expected
	commentFromDB, err := testCtx.dbMap.SelectStr("SELECT comment from blockedKeys WHERE keyHash = ?", duplicateKeySPKI)
	test.AssertNotError(t, err, "Failed to get comment from database")
	u, err := user.Current()
	test.AssertNotError(t, err, "Failed to get current user")
	expectedDBComment := fmt.Sprintf("%s: %s", u.Username, comment)
	test.AssertEquals(t, commentFromDB, expectedDBComment)
}

func TestPrivateKeyRevoke(t *testing.T) {
	testCtx := setup(t)
	defer testCtx.cleanUp()

	uniqueEntries, duplicateEntry := setupUniqueTestEntries(t)
	testCtx.createAndRegisterEntries(t, uniqueEntries)
	testCtx.createAndRegisterEntry(t, duplicateEntry)

	// Write the key contents of our duplicate entry to a temp file.
	duplicateKeyFile, err := os.CreateTemp("", "key")
	test.AssertNotError(t, err, "failed to create temp file")
	der, err := x509.MarshalPKCS8PrivateKey(duplicateEntry.testKey)
	test.AssertNotError(t, err, "failed to marshal testKey1 to DER")
	err = pem.Encode(duplicateKeyFile,
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		},
	)
	test.AssertNotError(t, err, "failed to PEM encode test key")
	test.AssertNotError(t, err, "failed to write to temp file")
	defer os.Remove(duplicateKeyFile.Name())

	// Get the SPKI hash for the provided keypair.
	duplicateKeySPKI, err := getPublicKeySPKIHash(&duplicateEntry.testKey.PublicKey)
	test.AssertNotError(t, err, "Failed to get SPKI hash for dupe.")

	// Query the 'keyHashToSerial' table for certificates with a matching SPKI
	// hash. We expect that since this key was re-used we'll find 2 matches.
	count, err := testCtx.revoker.countCertsMatchingSPKIHash(duplicateKeySPKI)
	test.AssertNotError(t, err, "countCertsMatchingSPKIHash for dupe failed")
	test.AssertEquals(t, count, 2)

	// With dryRun=true this should not revoke certificates or block issuance.
	err = privateKeyRevoke(&testCtx.revoker, true, "", count, duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")

	// Ensure that the key is not blocked, yet.
	keyExists, err := testCtx.revoker.spkiHashInBlockedKeys(duplicateKeySPKI)
	test.AssertNotError(t, err, "spkiHashInBlockedKeys failed for key that shouldn't be blocked yet")
	test.Assert(t, !keyExists, "SPKI hash should not be in blockedKeys")

	// With dryRun=false this should revoke matching certificates and block the key.
	comment := "key blocked as part of test"
	err = privateKeyRevoke(&testCtx.revoker, false, comment, count, duplicateKeyFile.Name())
	test.AssertNotError(t, err, "While attempting to block issuance for the provided key")

	// Ensure that the key is now blocked.
	keyExists, err = testCtx.revoker.spkiHashInBlockedKeys(duplicateKeySPKI)
	test.AssertNotError(t, err, "spkiHashInBlockedKeys failed for key that should now be blocked")
	test.Assert(t, keyExists, "SPKI hash should not be in blockedKeys")

	// Ensure that the comment was set as expected
	commentFromDB, err := testCtx.dbMap.SelectStr("SELECT comment from blockedKeys WHERE keyHash = ?", duplicateKeySPKI)
	test.AssertNotError(t, err, "Failed to get comment from database")
	u, err := user.Current()
	test.AssertNotError(t, err, "Failed to get current user")
	expectedDBComment := fmt.Sprintf("%s: %s", u.Username, comment)
	test.AssertEquals(t, commentFromDB, expectedDBComment)
}

type entry struct {
	jwk      string
	serial   *big.Int
	names    []string
	testKey  *rsa.PrivateKey
	regId    int64
	spkiHash []byte
}

func setupUniqueTestEntries(t *testing.T) ([]*entry, *entry) {
	t.Helper()

	// Unique keys for each of our test certificates.
	key1, err := rsa.GenerateKey(rand.Reader, 2048)
	test.AssertNotError(t, err, "Generating test key 1")
	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	test.AssertNotError(t, err, "Generating test key 2")
	key3, err := rsa.GenerateKey(rand.Reader, 2048)
	test.AssertNotError(t, err, "Generating test key 3")

	// Unique JWKs so we can register each of our entries.
	testJWK1 := `{"kty":"RSA","n":"yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ","e":"AQAB"}`
	testJWK2 := `{"kty":"RSA","n":"qnARLrT7Xz4gRcKyLdydmCr-ey9OuPImX4X40thk3on26FkMznR3fRjs66eLK7mmPcBZ6uOJseURU6wAaZNmemoYx1dMvqvWWIyiQleHSD7Q8vBrhR6uIoO4jAzJZR-ChzZuSDt7iHN-3xUVspu5XGwXU_MVJZshTwp4TaFx5elHIT_ObnTvTOU3Xhish07AbgZKmWsVbXh5s-CrIicU4OexJPgunWZ_YJJueOKmTvnLlTV4MzKR2oZlBKZ27S0-SfdV_QDx_ydle5oMAyKVtlAV35cyPMIsYNwgUGBCdY_2Uzi5eX0lTc7MPRwz6qR1kip-i59VcGcUQgqHV6Fyqw","e":"AQAB"}`
	testJWK3 := `{"kty":"RSA","n":"uTQER6vUA1RDixS8xsfCRiKUNGRzzyIK0MhbS2biClShbb0hSx2mPP7gBvis2lizZ9r-y9hL57kNQoYCKndOBg0FYsHzrQ3O9AcoV1z2Mq-XhHZbFrVYaXI0M3oY9BJCWog0dyi3XC0x8AxC1npd1U61cToHx-3uSvgZOuQA5ffEn5L38Dz1Ti7OV3E4XahnRJvejadUmTkki7phLBUXm5MnnyFm0CPpf6ApV7zhLjN5W-nV0WL17o7v8aDgV_t9nIdi1Y26c3PlCEtiVHZcebDH5F1Deta3oLLg9-g6rWnTqPbY3knffhp4m0scLD6e33k8MtzxDX_D7vHsg0_X1w","e":"AQAB"}`
	testJWK4 := `{"kty":"RSA","n":"qih-cx32M0wq8MhhN-kBi2xPE-wnw4_iIg1hWO5wtBfpt2PtWikgPuBT6jvK9oyQwAWbSfwqlVZatMPY_-3IyytMNb9R9OatNr6o5HROBoyZnDVSiC4iMRd7bRl_PWSIqj_MjhPNa9cYwBdW5iC3jM5TaOgmp0-YFm4tkLGirDcIBDkQYlnv9NKILvuwqkapZ7XBixeqdCcikUcTRXW5unqygO6bnapzw-YtPsPPlj4Ih3SvK4doyziPV96U8u5lbNYYEzYiW1mbu9n0KLvmKDikGcdOpf6-yRa_10kMZyYQatY1eclIKI0xb54kbluEl0GQDaL5FxLmiKeVnsapzw","e":"AQAB"}`
	return []*entry{
			{jwk: testJWK1, serial: big.NewInt(1), names: []string{"example-1337.com"}, testKey: key1},
			{jwk: testJWK2, serial: big.NewInt(2), names: []string{"example-1338.com"}, testKey: key2},
			{jwk: testJWK3, serial: big.NewInt(3), names: []string{"example-1339.com"}, testKey: key3},
		},
		&entry{jwk: testJWK4, serial: big.NewInt(4), names: []string{"example-1336.com"}, testKey: key1}
}

type testCtx struct {
	revoker revoker
	ssa     sapb.StorageAuthorityClient
	dbMap   *db.WrappedMap
	cleanUp func()
	issuer  *issuance.Certificate
	signer  crypto.Signer
	log     *blog.Mock
}

func (c testCtx) addRegistation(t *testing.T, names []string, jwk string) int64 {
	t.Helper()
	initialIP, err := net.ParseIP("127.0.0.1").MarshalText()
	test.AssertNotError(t, err, "Failed to create initialIP")

	reg := &corepb.Registration{
		Id:        1,
		Contact:   []string{fmt.Sprintf("hello@%s", names[0])},
		Key:       []byte(jwk),
		InitialIP: initialIP,
	}

	reg, err = c.ssa.NewRegistration(context.Background(), reg)
	test.AssertNotError(t, err, "Failed to store test registration")
	return reg.Id
}

func (c testCtx) addCertificate(t *testing.T, serial *big.Int, names []string, pubKey rsa.PublicKey, regId int64) *x509.Certificate {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"tests"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 1),
		DNSNames:     names,
	}

	rawCert, err := x509.CreateCertificate(rand.Reader, template, c.issuer.Certificate, &pubKey, c.signer)
	test.AssertNotError(t, err, "Failed to generate test cert")

	_, err = c.ssa.AddPrecertificate(
		context.Background(), &sapb.AddCertificateRequest{
			Der:      rawCert,
			RegID:    regId,
			Issued:   time.Now().UnixNano(),
			IssuerID: 1,
		},
	)
	test.AssertNotError(t, err, "Failed to add test precert")

	cert, err := x509.ParseCertificate(rawCert)
	test.AssertNotError(t, err, "Failed to parse test cert")
	return cert
}

func (c testCtx) createAndRegisterEntries(t *testing.T, entries []*entry) {
	t.Helper()
	for _, entry := range entries {
		c.createAndRegisterEntry(t, entry)
	}
}

func (c testCtx) createAndRegisterEntry(t *testing.T, e *entry) {
	t.Helper()
	e.regId = c.addRegistation(t, e.names, e.jwk)
	cert := c.addCertificate(t, e.serial, e.names, e.testKey.PublicKey, e.regId)
	var err error
	e.spkiHash, err = getPublicKeySPKIHash(cert.PublicKey)
	test.AssertNotError(t, err, "Failed to get SPKI hash")
}

func setup(t *testing.T) testCtx {
	t.Helper()
	log := blog.UseMock()
	fc := clock.NewFake()

	// Set some non-zero time for GRPC requests to be non-nil.
	fc.Set(time.Now())

	dbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	if err != nil {
		t.Fatalf("Failed to create dbMap: %s", err)
	}
	incidentsDbMap, err := sa.NewDbMap(vars.DBConnIncidents, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't create test dbMap")

	ssa, err := sa.NewSQLStorageAuthority(dbMap, dbMap, incidentsDbMap, 1, fc, log, metrics.NoopRegisterer)
	if err != nil {
		t.Fatalf("Failed to create SA: %s", err)
	}
	cleanUp := func() {
		test.ResetBoulderTestDatabase(t)
		test.ResetIncidentsTestDatabase(t)
	}

	issuer, err := issuance.LoadCertificate("../../test/hierarchy/int-r3.cert.pem")
	test.AssertNotError(t, err, "Failed to load test issuer")

	signer, err := test.LoadSigner("../../test/hierarchy/int-r3.key.pem")
	test.AssertNotError(t, err, "Failed to load test signer")

	ra := ra.NewRegistrationAuthorityImpl(
		fc,
		log,
		metrics.NoopRegisterer,
		1,
		goodkey.KeyPolicy{},
		100,
		true,
		300*24*time.Hour,
		7*24*time.Hour,
		nil,
		nil,
		0,
		nil,
		&mockPurger{},
		[]*issuance.Certificate{issuer},
	)
	ra.SA = isa.SA{Impl: ssa}
	ra.OCSP = &mockOCSPA{}
	rac := ira.RA{Impl: ra}

	return testCtx{
		revoker: revoker{rac, isa.SA{Impl: ssa}, dbMap, fc, log},
		ssa:     isa.SA{Impl: ssa},
		dbMap:   dbMap,
		cleanUp: cleanUp,
		issuer:  issuer,
		signer:  signer,
		log:     log,
	}
}
