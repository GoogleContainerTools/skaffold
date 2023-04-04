package notmain

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/asn1"
	"encoding/pem"
	"log"
	"math/big"
	mrand "math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmhodges/clock"

	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/ctpolicy/loglist"
	"github.com/letsencrypt/boulder/goodkey"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/policy"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/sa/satest"
	"github.com/letsencrypt/boulder/test"
	isa "github.com/letsencrypt/boulder/test/inmem/sa"
	"github.com/letsencrypt/boulder/test/vars"
)

var (
	testValidityDuration  = 24 * 90 * time.Hour
	testValidityDurations = map[time.Duration]bool{testValidityDuration: true}
	pa                    *policy.AuthorityImpl
	kp                    goodkey.KeyPolicy
)

func init() {
	var err error
	pa, err = policy.New(map[core.AcmeChallenge]bool{}, blog.NewMock())
	if err != nil {
		log.Fatal(err)
	}
	err = pa.SetHostnamePolicyFile("../../test/hostname-policy.yaml")
	if err != nil {
		log.Fatal(err)
	}
	kp, err = goodkey.NewKeyPolicy(&goodkey.Config{FermatRounds: 100}, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkCheckCert(b *testing.B) {
	checker := newChecker(nil, clock.New(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	testKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	expiry := time.Now().AddDate(0, 0, 1)
	serial := big.NewInt(1337)
	rawCert := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		NotAfter:     expiry,
		DNSNames:     []string{"example-a.com"},
		SerialNumber: serial,
	}
	certDer, _ := x509.CreateCertificate(rand.Reader, &rawCert, &rawCert, &testKey.PublicKey, testKey)
	cert := core.Certificate{
		Serial:  core.SerialToString(serial),
		Digest:  core.Fingerprint256(certDer),
		DER:     certDer,
		Issued:  time.Now(),
		Expires: expiry,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.checkCert(cert, nil)
	}
}

func TestCheckWildcardCert(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	saCleanup := test.ResetBoulderTestDatabase(t)
	defer func() {
		saCleanup()
	}()

	testKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	fc := clock.NewFake()
	checker := newChecker(saDbMap, fc, pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	issued := checker.clock.Now().Add(-time.Minute)
	goodExpiry := issued.Add(testValidityDuration - time.Second)
	serial := big.NewInt(1337)

	wildcardCert := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "*.example.com",
		},
		NotBefore:             issued,
		NotAfter:              goodExpiry,
		DNSNames:              []string{"*.example.com"},
		SerialNumber:          serial,
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		OCSPServer:            []string{"http://example.com/ocsp"},
		IssuingCertificateURL: []string{"http://example.com/cert"},
	}
	wildcardCertDer, err := x509.CreateCertificate(rand.Reader, &wildcardCert, &wildcardCert, &testKey.PublicKey, testKey)
	test.AssertNotError(t, err, "Couldn't create certificate")
	parsed, err := x509.ParseCertificate(wildcardCertDer)
	test.AssertNotError(t, err, "Couldn't parse created certificate")
	cert := core.Certificate{
		Serial:  core.SerialToString(serial),
		Digest:  core.Fingerprint256(wildcardCertDer),
		Expires: parsed.NotAfter,
		Issued:  parsed.NotBefore,
		DER:     wildcardCertDer,
	}
	_, problems := checker.checkCert(cert, nil)
	for _, p := range problems {
		t.Errorf(p)
	}
}

func TestCheckCertReturnsDNSNames(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	saCleanup := test.ResetBoulderTestDatabase(t)
	defer func() {
		saCleanup()
	}()
	checker := newChecker(saDbMap, clock.NewFake(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())

	certPEM, err := os.ReadFile("testdata/quite_invalid.pem")
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to parse cert PEM")
	}

	cert := core.Certificate{
		Serial:  "00000000000",
		Digest:  core.Fingerprint256(block.Bytes),
		Expires: time.Now().Add(time.Hour),
		Issued:  time.Now(),
		DER:     block.Bytes,
	}

	names, problems := checker.checkCert(cert, nil)
	if !reflect.DeepEqual(names, []string{"quite_invalid.com", "al--so--wr--ong.com"}) {
		t.Errorf("didn't get expected DNS names. other problems: %s", strings.Join(problems, "\n"))
	}
}

type keyGen interface {
	genKey() (crypto.Signer, error)
}

type ecP256Generator struct{}

func (*ecP256Generator) genKey() (crypto.Signer, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

type rsa2048Generator struct{}

func (*rsa2048Generator) genKey() (crypto.Signer, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func TestCheckCert(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	saCleanup := test.ResetBoulderTestDatabase(t)
	defer func() {
		saCleanup()
	}()

	testCases := []struct {
		name string
		key  keyGen
	}{
		{
			name: "RSA 2048 key",
			key:  &rsa2048Generator{},
		},
		{
			name: "ECDSA P256 key",
			key:  &ecP256Generator{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testKey, _ := tc.key.genKey()

			checker := newChecker(saDbMap, clock.NewFake(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())

			// Create a RFC 7633 OCSP Must Staple Extension.
			// OID 1.3.6.1.5.5.7.1.24
			ocspMustStaple := pkix.Extension{
				Id:       asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 1, 24},
				Critical: false,
				Value:    []uint8{0x30, 0x3, 0x2, 0x1, 0x5},
			}

			// Create a made up PKIX extension
			imaginaryExtension := pkix.Extension{
				Id:       asn1.ObjectIdentifier{1, 3, 3, 7},
				Critical: false,
				Value:    []uint8{0xC0, 0xFF, 0xEE},
			}

			issued := checker.clock.Now().Add(-time.Minute)
			goodExpiry := issued.Add(testValidityDuration - time.Second)
			serial := big.NewInt(1337)
			longName := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeexample.com"
			rawCert := x509.Certificate{
				Subject: pkix.Name{
					CommonName: longName,
				},
				NotBefore: issued,
				NotAfter:  goodExpiry.AddDate(0, 0, 1), // Period too long
				DNSNames: []string{
					// longName should be flagged along with the long CN
					longName,
					"example-a.com",
					"foodnotbombs.mil",
					// `dev-myqnapcloud.com` is included because it is an exact private
					// entry on the public suffix list
					"dev-myqnapcloud.com",
				},
				SerialNumber:          serial,
				BasicConstraintsValid: false,
				ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				KeyUsage:              x509.KeyUsageDigitalSignature,
				OCSPServer:            []string{"http://example.com/ocsp"},
				IssuingCertificateURL: []string{"http://example.com/cert"},
				ExtraExtensions:       []pkix.Extension{ocspMustStaple, imaginaryExtension},
			}
			brokenCertDer, err := x509.CreateCertificate(rand.Reader, &rawCert, &rawCert, testKey.Public(), testKey)
			test.AssertNotError(t, err, "Couldn't create certificate")
			// Problems
			//   Digest doesn't match
			//   Serial doesn't match
			//   Expiry doesn't match
			//   Issued doesn't match
			cert := core.Certificate{
				Serial:  "8485f2687eba29ad455ae4e31c8679206fec",
				DER:     brokenCertDer,
				Issued:  issued.Add(12 * time.Hour),
				Expires: goodExpiry.AddDate(0, 0, 2), // Expiration doesn't match
			}

			_, problems := checker.checkCert(cert, nil)

			problemsMap := map[string]int{
				"Stored digest doesn't match certificate digest":                            1,
				"Stored serial doesn't match certificate serial":                            1,
				"Stored expiration doesn't match certificate NotAfter":                      1,
				"Certificate doesn't have basic constraints set":                            1,
				"Certificate has unacceptable validity period":                              1,
				"Stored issuance date is outside of 6 hour window of certificate NotBefore": 1,
				"Certificate has incorrect key usage extensions":                            1,
				"Certificate has common name >64 characters long (65)":                      1,
				"Certificate contains an unexpected extension: 1.3.3.7":                     1,
			}
			for _, p := range problems {
				_, ok := problemsMap[p]
				if !ok {
					t.Errorf("Found unexpected problem '%s'.", p)
				}
				delete(problemsMap, p)
			}
			for k := range problemsMap {
				t.Errorf("Expected problem but didn't find it: '%s'.", k)
			}

			// Same settings as above, but the stored serial number in the DB is invalid.
			cert.Serial = "not valid"
			_, problems = checker.checkCert(cert, nil)
			foundInvalidSerialProblem := false
			for _, p := range problems {
				if p == "Stored serial is invalid" {
					foundInvalidSerialProblem = true
				}
			}
			test.Assert(t, foundInvalidSerialProblem, "Invalid certificate serial number in DB did not trigger problem.")

			// Fix the problems
			rawCert.Subject.CommonName = "example-a.com"
			rawCert.DNSNames = []string{"example-a.com"}
			rawCert.NotAfter = goodExpiry
			rawCert.BasicConstraintsValid = true
			rawCert.ExtraExtensions = []pkix.Extension{ocspMustStaple}
			rawCert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
			goodCertDer, err := x509.CreateCertificate(rand.Reader, &rawCert, &rawCert, testKey.Public(), testKey)
			test.AssertNotError(t, err, "Couldn't create certificate")
			parsed, err := x509.ParseCertificate(goodCertDer)
			test.AssertNotError(t, err, "Couldn't parse created certificate")
			cert.Serial = core.SerialToString(serial)
			cert.Digest = core.Fingerprint256(goodCertDer)
			cert.DER = goodCertDer
			cert.Expires = parsed.NotAfter
			cert.Issued = parsed.NotBefore
			_, problems = checker.checkCert(cert, nil)
			test.AssertEquals(t, len(problems), 0)
		})
	}
}

func TestGetAndProcessCerts(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	fc := clock.NewFake()
	fc.Set(fc.Now().Add(time.Hour))

	checker := newChecker(saDbMap, fc, pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	sa, err := sa.NewSQLStorageAuthority(saDbMap, saDbMap, nil, 1, fc, blog.NewMock(), metrics.NoopRegisterer)
	test.AssertNotError(t, err, "Couldn't create SA to insert certificates")
	saCleanUp := test.ResetBoulderTestDatabase(t)
	defer func() {
		saCleanUp()
	}()

	testKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	// Problems
	//   Expiry period is too long
	rawCert := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "not-blacklisted.com",
		},
		BasicConstraintsValid: true,
		DNSNames:              []string{"not-blacklisted.com"},
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	reg := satest.CreateWorkingRegistration(t, isa.SA{Impl: sa})
	test.AssertNotError(t, err, "Couldn't create registration")
	for i := int64(0); i < 5; i++ {
		rawCert.SerialNumber = big.NewInt(mrand.Int63())
		certDER, err := x509.CreateCertificate(rand.Reader, &rawCert, &rawCert, &testKey.PublicKey, testKey)
		test.AssertNotError(t, err, "Couldn't create certificate")
		_, err = sa.AddCertificate(context.Background(), &sapb.AddCertificateRequest{
			Der:    certDER,
			RegID:  reg.Id,
			Issued: fc.Now().UnixNano(),
		})
		test.AssertNotError(t, err, "Couldn't add certificate")
	}

	batchSize = 2
	err = checker.getCerts(false)
	test.AssertNotError(t, err, "Failed to retrieve certificates")
	test.AssertEquals(t, len(checker.certs), 5)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	checker.processCerts(wg, false, nil)
	test.AssertEquals(t, checker.issuedReport.BadCerts, int64(5))
	test.AssertEquals(t, len(checker.issuedReport.Entries), 5)
}

// mismatchedCountDB is a certDB implementation for `getCerts` that returns one
// high value when asked how many rows there are, and then returns nothing when
// asked for the actual rows.
type mismatchedCountDB struct{}

// `getCerts` calls `SelectInt` first to determine how many rows there are
// matching the `getCertsCountQuery` criteria. For this mock we return
// a non-zero number
func (db mismatchedCountDB) SelectNullInt(_ string, _ ...interface{}) (sql.NullInt64, error) {
	return sql.NullInt64{
			Int64: 99999,
			Valid: true,
		},
		nil
}

// `getCerts` then calls `Select` to retrieve the Certificate rows. We pull
// a dastardly switch-a-roo here and return an empty set
func (db mismatchedCountDB) Select(output interface{}, _ string, _ ...interface{}) ([]interface{}, error) {
	// But actually return nothing
	outputPtr, _ := output.(*[]sa.CertWithID)
	*outputPtr = []sa.CertWithID{}
	return nil, nil
}

/*
 * In Boulder #2004[0] we identified that there is a race in `getCerts`
 * between the first call to `SelectOne` to identify how many rows there are,
 * and the subsequent call to `Select` to get the actual rows in batches. This
 * manifests in an index out of range panic where the cert checker thinks there
 * are more rows than there are and indexes into an empty set of certificates to
 * update the lastSerial field of the query `args`. This has been fixed by
 * adding a len() check in the inner `getCerts` loop that processes the certs
 * one batch at a time.
 *
 * TestGetCertsEmptyResults tests the fix remains in place by using a mock that
 * exploits this corner case deliberately. The `mismatchedCountDB` mock (defined
 * above) will return a high count for the `SelectOne` call, but an empty slice
 * for the `Select` call. Without the fix in place this reliably produced the
 * "index out of range" panic from #2004. With the fix in place the test passes.
 *
 * 0: https://github.com/letsencrypt/boulder/issues/2004
 */
func TestGetCertsEmptyResults(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	checker := newChecker(saDbMap, clock.NewFake(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	checker.dbMap = mismatchedCountDB{}

	batchSize = 3
	err = checker.getCerts(false)
	test.AssertNotError(t, err, "Failed to retrieve certificates")
}

// emptyDB is a certDB object with methods used for testing that 'null'
// responses received from the database are handled properly.
type emptyDB struct {
	certDB
}

// SelectNullInt is a method that returns a false sql.NullInt64 struct to
// mock a null DB response
func (db emptyDB) SelectNullInt(_ string, _ ...interface{}) (sql.NullInt64, error) {
	return sql.NullInt64{Valid: false},
		nil
}

// TestGetCertsNullResults tests that a null response from the database will
// be handled properly. It uses the emptyDB above to mock the response
// expected if the DB finds no certificates to match the SELECT query and
// should return an error.
func TestGetCertsNullResults(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	checker := newChecker(saDbMap, clock.NewFake(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	checker.dbMap = emptyDB{}

	err = checker.getCerts(false)
	test.AssertError(t, err, "Should have gotten error from empty DB")
}

func TestSaveReport(t *testing.T) {
	r := report{
		begin:     time.Time{},
		end:       time.Time{},
		GoodCerts: 2,
		BadCerts:  1,
		Entries: map[string]reportEntry{
			"020000000000004b475da49b91da5c17": {
				Valid: true,
			},
			"020000000000004d1613e581432cba7e": {
				Valid: true,
			},
			"020000000000004e402bc21035c6634a": {
				Valid:    false,
				Problems: []string{"None really..."},
			},
		},
	}

	err := r.dump()
	test.AssertNotError(t, err, "Failed to dump results")
}

func TestIsForbiddenDomain(t *testing.T) {
	// Note: These testcases are not an exhaustive representation of domains
	// Boulder won't issue for, but are instead testing the defense-in-depth
	// `isForbiddenDomain` function called *after* the PA has vetted the name
	// against the complex hostname policy file.
	testcases := []struct {
		Name     string
		Expected bool
	}{
		/* Expected to be forbidden test cases */
		// Whitespace only
		{Name: "", Expected: true},
		{Name: "   ", Expected: true},
		// Anything .local
		{Name: "yokel.local", Expected: true},
		{Name: "off.on.remote.local", Expected: true},
		{Name: ".local", Expected: true},
		// Localhost is verboten
		{Name: "localhost", Expected: true},
		// Anything .localhost
		{Name: ".localhost", Expected: true},
		{Name: "local.localhost", Expected: true},
		{Name: "extremely.local.localhost", Expected: true},

		/* Expected to be allowed test cases */
		{Name: "ok.computer.com", Expected: false},
		{Name: "ok.millionaires", Expected: false},
		{Name: "ok.milly", Expected: false},
		{Name: "ok", Expected: false},
		{Name: "nearby.locals", Expected: false},
		{Name: "yocalhost", Expected: false},
		{Name: "jokes.yocalhost", Expected: false},
	}

	for _, tc := range testcases {
		result, _ := isForbiddenDomain(tc.Name)
		test.AssertEquals(t, result, tc.Expected)
	}
}

func TestIgnoredLint(t *testing.T) {
	saDbMap, err := sa.NewDbMap(vars.DBConnSA, sa.DbSettings{})
	test.AssertNotError(t, err, "Couldn't connect to database")
	saCleanup := test.ResetBoulderTestDatabase(t)
	defer func() {
		saCleanup()
	}()

	err = loglist.InitLintList("../../test/ct-test-srv/log_list.json")
	test.AssertNotError(t, err, "failed to load ct log list")
	testKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	checker := newChecker(saDbMap, clock.NewFake(), pa, kp, time.Hour, testValidityDurations, blog.NewMock())
	serial := big.NewInt(1337)

	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "CPU's Cool CA",
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(testValidityDuration - time.Second),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		PolicyIdentifiers: []asn1.ObjectIdentifier{
			{1, 2, 3},
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IssuingCertificateURL: []string{"http://ca.cpu"},
		SubjectKeyId:          []byte("foobar"),
	}

	// Create a self-signed issuer certificate to use
	issuerDer, err := x509.CreateCertificate(rand.Reader, template, template, testKey.Public(), testKey)
	test.AssertNotError(t, err, "failed to create self-signed issuer cert")
	issuerCert, err := x509.ParseCertificate(issuerDer)
	test.AssertNotError(t, err, "failed to parse self-signed issuer cert")

	// Reconfigure the template for an EE cert with a Subj. CN
	serial = big.NewInt(1338)
	template.SerialNumber = serial
	template.Subject.CommonName = "zombo.com"
	template.DNSNames = []string{"zombo.com"}
	template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	template.IsCA = false

	subjectCertDer, err := x509.CreateCertificate(rand.Reader, template, issuerCert, testKey.Public(), testKey)
	test.AssertNotError(t, err, "failed to create EE cert")
	subjectCert, err := x509.ParseCertificate(subjectCertDer)
	test.AssertNotError(t, err, "failed to parse EE cert")

	cert := core.Certificate{
		Serial:  core.SerialToString(serial),
		DER:     subjectCertDer,
		Digest:  core.Fingerprint256(subjectCertDer),
		Issued:  subjectCert.NotBefore,
		Expires: subjectCert.NotAfter,
	}

	// Without any ignored lints we expect one error level result due to the
	// missing OCSP url in the template.
	expectedProblems := []string{
		"zlint error: e_sub_cert_aia_does_not_contain_ocsp_url",
		"zlint info: n_subject_common_name_included",
		"zlint info: w_ct_sct_policy_count_unsatisfied Certificate had 0 embedded SCTs. Browser policy may require 2 for this certificate.",
		"zlint error: e_scts_from_same_operator Certificate had too few embedded SCTs; browser policy requires 2.",
	}
	sort.Strings(expectedProblems)

	// Check the certificate with a nil ignore map. This should return the
	// expected zlint problems.
	_, problems := checker.checkCert(cert, nil)
	sort.Strings(problems)
	test.AssertDeepEquals(t, problems, expectedProblems)

	// Check the certificate again with an ignore map that excludes the affected
	// lints. This should return no problems.
	_, problems = checker.checkCert(cert, map[string]bool{
		"e_sub_cert_aia_does_not_contain_ocsp_url": true,
		"n_subject_common_name_included":           true,
		"w_ct_sct_policy_count_unsatisfied":        true,
		"e_scts_from_same_operator":                true,
	})
	test.AssertEquals(t, len(problems), 0)
}
